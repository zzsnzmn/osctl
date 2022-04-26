// Copyright 2019 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package encoder is a widget that displays the progress of an operation as a
// partial or full circle.
package encoder

import (
	"errors"
	"fmt"
	"image"
	"log"
	"math"
	"sync"

	"github.com/hypebeast/go-osc/osc"
	"github.com/mum4k/termdash/align"
	"github.com/mum4k/termdash/mouse"
	"github.com/mum4k/termdash/private/alignfor"
	"github.com/mum4k/termdash/private/area"
	"github.com/mum4k/termdash/private/canvas"
	"github.com/mum4k/termdash/private/canvas/braille"
	"github.com/mum4k/termdash/private/draw"
	"github.com/mum4k/termdash/private/runewidth"
	"github.com/mum4k/termdash/terminal/terminalapi"
	"github.com/mum4k/termdash/widgetapi"
)

// Encoder displays the progress of an operation by filling a partial circle and
// eventually by completing a full circle. The circle can have a "center" in the
// middle, which is where the name comes from.
//
// Implements widgetapi.Widget. This object is thread-safe.
type Encoder struct {
	// current is the current progress that will be drawn.
	current int
	// the total provided by the caller
	total int
	// angle is the value that represents the angle in radians (-360, 360)
	angle int
	// mu protects the Encoder.
	mu sync.Mutex

	// opts are the provided options.
	opts *options

	oscPort  int
	oscAddr  string
	oscRoute string
}

// New returns a new Encoder.
func New(opts ...Option) (*Encoder, error) {
	opt := newOptions()
	for _, o := range opts {
		o.set(opt)
	}
	if err := opt.validate(); err != nil {
		return nil, err
	}
	return &Encoder{
		oscRoute: opt.oscRoute,
		oscPort:  opt.oscPort,
		oscAddr:  opt.oscAddr,
		angle:    opt.startAngle,
		opts:     opt,
	}, nil
}

// Percent sets the current progress in percentage.
// The provided value must be between 0 and 100.
// Provided options override values set when New() was called.
// TODO: make this degrees?:
func (d *Encoder) Percent(p int, opts ...Option) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if p < 0 || p > 100 {
		return fmt.Errorf("invalid percentage, p(%d) must be 0 <= p <= 100", p)
	}

	for _, opt := range opts {
		opt.set(d.opts)
	}
	if err := d.opts.validate(); err != nil {
		return err
	}

	//d.current = 6
	d.total = 100
	return nil
}

// progressText returns the textual representation of the current progress.
func (d *Encoder) progressText() string {
	return fmt.Sprintf("%d%%", d.current)
}

// centerRadius calculates the radius of the "center" in the encoder.
// Returns zero if no center should be drawn.
func (d *Encoder) centerRadius(encoderRadius int) int {
	r := int(math.Round(float64(encoderRadius) / 100 * float64(d.opts.centerPercent)))
	if r < 2 { // Smallest possible circle radius.
		return 0
	}
	return r
}

// drawText draws the text label showing the progress.
// The text is only drawn if the radius of the encoder "center" is large enough to
// accommodate it.
// The mid point addresses coordinates in pixels on a braille canvas.
func (d *Encoder) drawText(cvs *canvas.Canvas, mid image.Point, centerR int) error {
	cells, first := availableCells(mid, centerR)
	t := d.progressText()
	needCells := runewidth.StringWidth(t)
	if cells < needCells {
		return nil
	}

	ar := image.Rect(first.X, first.Y, first.X+cells+2, first.Y+1)
	start, err := alignfor.Text(ar, t, align.HorizontalCenter, align.VerticalMiddle)
	if err != nil {
		return fmt.Errorf("alignfor.Text => %v", err)
	}
	if err := draw.Text(cvs, t, start, draw.TextMaxX(start.X+needCells), draw.TextCellOpts(d.opts.textCellOpts...)); err != nil {
		return fmt.Errorf("draw.Text => %v", err)
	}
	return nil
}

// drawLabel draws the text label in the area.
func (d *Encoder) drawLabel(cvs *canvas.Canvas, labelAr image.Rectangle) error {
	start, err := alignfor.Text(labelAr, d.opts.label, d.opts.labelAlign, align.VerticalBottom)
	if err != nil {
		return err
	}
	return draw.Text(
		cvs, d.opts.label, start,
		draw.TextOverrunMode(draw.OverrunModeThreeDot),
		draw.TextMaxX(labelAr.Max.X),
		draw.TextCellOpts(d.opts.labelCellOpts...),
	)
}

// Draw draws the Encoder widget onto the canvas.
// Implements widgetapi.Widget.Draw.
func (d *Encoder) Draw(cvs *canvas.Canvas, _ *widgetapi.Meta) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// TODO: make this aware of the angle
	startA, endA := startEndAngles(d.current, d.total, d.angle, d.opts.direction)

	var encoderAr, labelAr image.Rectangle
	if len(d.opts.label) > 0 {
		d, l, err := encoderAndLabel(cvs.Area())
		if err != nil {
			return err
		}
		encoderAr = d
		labelAr = l

	} else {
		encoderAr = cvs.Area()
	}

	if encoderAr.Dx() < minSize.X || encoderAr.Dy() < minSize.Y {
		// Reserving area for the label might have resulted in encoderAr being
		// too small.
		return draw.ResizeNeeded(cvs)
	}

	bc, err := braille.New(encoderAr)
	if err != nil {
		return fmt.Errorf("braille.New => %v", err)
	}

	mid, r := midAndRadius(bc.Area())
	if err := draw.BrailleCircle(bc, mid, r,
		draw.BrailleCircleFilled(),
		draw.BrailleCircleArcOnly(startA, endA),
		draw.BrailleCircleCellOpts(d.opts.cellOpts...),
	); err != nil {
		return fmt.Errorf("failed to draw the outer circle: %v", err)
	}

	centerR := d.centerRadius(r)
	if centerR != 0 {
		if err := draw.BrailleCircle(bc, mid, centerR,
			draw.BrailleCircleFilled(),
			draw.BrailleCircleClearPixels(),
		); err != nil {
			return fmt.Errorf("failed to draw the outer circle: %v", err)
		}
	}
	if err := bc.CopyTo(cvs); err != nil {
		return err
	}

	if !d.opts.hideTextProgress {
		if err := d.drawText(cvs, mid, centerR); err != nil {
			return err
		}
	}

	if !labelAr.Empty() {
		if err := d.drawLabel(cvs, labelAr); err != nil {
			return err
		}
	}
	return nil
}

// Keyboard input isn't supported on the Encoder widget.
func (*Encoder) Keyboard(_ *terminalapi.Keyboard, _ *widgetapi.EventMeta) error {
	return errors.New("the Encoder widget doesn't support keyboard events")
}

// Mouse input isn't supported on the Encoder widget.
func (d *Encoder) Mouse(m *terminalapi.Mouse, _ *widgetapi.EventMeta) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	client := osc.NewClient(d.oscAddr, d.oscPort)
	msg := osc.NewMessage(d.oscRoute)

	if m.Button == mouse.ButtonWheelDown {
		d.current = (d.current + 1) % d.total
		msg.Append(int32(1))
	}
	if m.Button == mouse.ButtonWheelUp {
		d.current = (d.current - 1) % d.total
		msg.Append(int32(-1))
	}

	err := client.Send(msg)
	if err != nil {
		log.Printf("error sending osc message")
	}

	return nil
}

// minSize is the smallest area we can draw encoder on.
var minSize = image.Point{3, 3}

// Options implements widgetapi.Widget.Options.
func (d *Encoder) Options() widgetapi.Options {
	return widgetapi.Options{
		// We are drawing a circle, ensure equal ratio of rows and columns.
		// This is adjusted for the inequality of the braille canvas.
		Ratio: image.Point{braille.RowMult, braille.ColMult},

		// The smallest circle that "looks" like a circle on the canvas.
		MinimumSize:  minSize,
		WantKeyboard: widgetapi.KeyScopeNone,
		WantMouse:    widgetapi.MouseScopeWidget,
	}
}

// encoderAndLabel splits the canvas area into an area for the encoder and an
// area under the encoder for the text label.
func encoderAndLabel(cvsAr image.Rectangle) (donAr, labelAr image.Rectangle, err error) {
	height := cvsAr.Dy()
	// Two lines for the text label at the bottom.
	// One for the text itself and one for visual space between the encoder and
	// the label.
	donAr, labelAr, err = area.HSplitCells(cvsAr, height-2)
	if err != nil {
		return image.ZR, image.ZR, err
	}
	return donAr, labelAr, nil
}

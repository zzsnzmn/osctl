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

package encoder

// options.go contains configurable options for Encoder.

import (
	"fmt"

	"github.com/mum4k/termdash/align"
	"github.com/mum4k/termdash/cell"
)

// Option is used to provide options.
type Option interface {
	// set sets the provided option.
	set(*options)
}

// option implements Option.
type option func(*options)

// set implements Option.set.
func (o option) set(opts *options) {
	o(opts)
}

// options holds the provided options.
type options struct {
	centerPercent int
	hideTextProgress bool

	textCellOpts []cell.Option
	cellOpts     []cell.Option

	labelCellOpts []cell.Option
	labelAlign    align.Horizontal
	label         string

	// The angle in degrees that represents 0 and 100% of the progress.
	startAngle int
	// The direction in which the encoder completes as progress increases.
	// Positive for counter-clockwise, negative for clockwise.
	direction int

	// TODO: add osc fields here
	lowerBound int
	upperBound int

	oscRoute string
	oscAddr string
	oscPort int

}

// validate validates the provided options.
func (o *options) validate() error {
	if min, max := 0, 100; o.centerPercent < min || o.centerPercent > max {
		return fmt.Errorf("invalid hole (lol) percent %d, must be in range %d <= p <= %d", o.centerPercent, min, max)
	}

	if min, max := 0, 360; o.startAngle < min || o.startAngle >= max {
		return fmt.Errorf("invalid start angle %d, must be in range %d <= angle < %d", o.startAngle, min, max)
	}

	if o.oscRoute == "" {
		return fmt.Errorf("invalid osc route %s", o.oscRoute)
	}


	return nil
}

// newOptions returns options with the default values set.
func newOptions() *options {
	return &options{
		centerPercent: DefaultCenterPercent,
		startAngle:       DefaultStartAngle,
		direction:        -1,
		textCellOpts: []cell.Option{
			cell.FgColor(cell.ColorDefault),
			cell.BgColor(cell.ColorDefault),
		},
		labelAlign: DefaultLabelAlign,
		oscAddr: "localhost", // make this configured...
		oscPort: 10111,
		oscRoute: "",
	}
}

// DefaultHolePercent is the default value for the HolePercent
// option.
const DefaultCenterPercent = 15

// HolePercent sets the size of the "hole" inside the encoder as a
// percentage of the encoder's radius.
// Setting this to zero disables the hole so that the encoder will become just a
// circle. Valid range is 0 <= p <= 100.
func CenterPercent(p int) Option {
	return option(func(opts *options) {
		opts.centerPercent = p
	})
}

// ShowTextProgress configures the Gauge so that it also displays a text
// enumerating the progress. This is the default behavior.
// If the progress is set by a call to Percent(), the displayed text will show
// the percentage, e.g. "50%". If the progress is set by a call to Absolute(),
// the displayed text will those the absolute numbers, e.g. "5/10".
//
// The progress is only displayed if there is enough space for it in the middle
// of the drawn encoder.
//
// Providing this option also sets HolePercent to its default value.
func ShowTextProgress() Option {
	return option(func(opts *options) {
		opts.hideTextProgress = false
	})
}

// HideTextProgress disables the display of a text enumerating the progress.
func HideTextProgress() Option {
	return option(func(opts *options) {
		opts.hideTextProgress = true
	})
}

// TextCellOpts sets cell options on cells that contain the displayed text
// progress.
func TextCellOpts(cOpts ...cell.Option) Option {
	return option(func(opts *options) {
		opts.textCellOpts = cOpts
	})
}

// CellOpts sets cell options on cells that contain the encoder.
func CellOpts(cOpts ...cell.Option) Option {
	return option(func(opts *options) {
		opts.cellOpts = cOpts
	})
}

// DefaultStartAngle is the default value for the StartAngle option.
const DefaultStartAngle = 90

// StartAngle sets the starting angle in degrees, i.e. the point that will
// represent both 0% and 100% of progress.
// Valid values are in range 0 <= angle < 360.
// Angles start at the X axis and grow counter-clockwise.
func StartAngle(angle int) Option {
	return option(func(opts *options) {
		opts.startAngle = angle
	})
}

// Clockwise sets the encoder widget for a progression in the clockwise
// direction. This is the default option.
func Clockwise() Option {
	return option(func(opts *options) {
		opts.direction = -1
	})
}

// CounterClockwise sets the encoder widget for a progression in the counter-clockwise
// direction.
func CounterClockwise() Option {
	return option(func(opts *options) {
		opts.direction = 1
	})
}

// Label sets a text label to be displayed under the encoder.
func Label(text string, cOpts ...cell.Option) Option {
	return option(func(opts *options) {
		opts.label = text
		opts.labelCellOpts = cOpts
	})
}

func OscRoute(route, addr string, port int) Option {
	return option(func(opts *options) {
		opts.oscRoute = route
		opts.oscAddr = addr
		opts.oscPort = port
	})
}

// DefaultLabelAlign is the default value for the LabelAlign option.
const DefaultLabelAlign = align.HorizontalCenter

// LabelAlign sets the alignment of the label under the encoder.
func LabelAlign(la align.Horizontal) Option {
	return option(func(opts *options) {
		opts.labelAlign = la
	})
}

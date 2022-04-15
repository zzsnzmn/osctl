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

// Binary encoderdemo displays a couple of encoder widgets.
// Exist when 'q' is pressed.
package main

import (
	"context"
	"time"
	//"log"
	"fmt"

	"github.com/hypebeast/go-osc/osc"
	"github.com/mum4k/termdash"
	"github.com/mum4k/termdash/align"
	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/container"
	"github.com/mum4k/termdash/linestyle"
	"github.com/mum4k/termdash/terminal/tcell"
	"github.com/mum4k/termdash/terminal/terminalapi"
	"github.com/mum4k/termdash/widgets/button"
	"github.com/mum4k/termdash/widgets/segmentdisplay"
	"github.com/mum4k/termdash/widgets/text"

	"github.com/zzsnzmn/osctl/internal/encoder"
	//"github.com/zzsnzmn/osctl/internal/screen"
)

// playType indicates how to play a encoder.
type playType int

const (
	playTypePercent playType = iota
	playTypeAbsolute
)

// drawEncoder continuously changes the displayed percent value on the encoder by the
// step once every delay. Exits when the context expires.
func drawEncoder(ctx context.Context, d *encoder.Encoder, start, step int, delay time.Duration, pt playType) {
	progress := start
	mult := 1

	ticker := time.NewTicker(delay)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			switch pt {
			case playTypePercent:
				if err := d.Percent(progress); err != nil {
					panic(err)
				}
			case playTypeAbsolute:
				if err := d.Absolute(progress, 100); err != nil {
					panic(err)
				}
			}

			progress += step * mult
			if progress > 100 || 100-progress < step {
				progress = 100
			} else if progress < 0 || progress < step {
				progress = 0
			}

			if progress == 100 {
				mult = -1
			} else if progress == 0 {
				mult = 1
			}

		case <-ctx.Done():
			return
		}
	}
}

// drawScreen writes lines of text to the text widget every delay.
// Exits when the context expires.
func drawScreen(ctx context.Context, t *text.Text, delay time.Duration) {
	ticker := time.NewTicker(delay)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			//s := screen.DisplayBuffer()
			s := ""
			if s == "" {
				continue
			}
			if err := t.Write(s, text.WriteReplace()); err != nil {
				panic(err)
			}

		case <-ctx.Done():
			return
		}
	}
}

func main() {
	t, err := tcell.New()
	if err != nil {
		panic(err)
	}
	defer t.Close()
	// TODO: read arg here to set hostname
	// TODO: read arg here to set port
	// TODO: read arg here to set range
	// TODO: read arg here to set osc msg type
	// TODO: read arg here to set osc msg route

	oscPort := 10111
	oscAddr := "localhost"
	//oscAddr := "68.183.203.181"
	ctx, cancel := context.WithCancel(context.Background())
	encoder1, err := encoder.New(
		encoder.CellOpts(cell.FgColor(cell.ColorGreen)),
		encoder.Label("E1", cell.FgColor(cell.ColorGreen)),
		encoder.OscRoute("/remote/enc/1", oscAddr, 10111),
	)
	if err != nil {
		panic(err)
	}
	// TODO: rename redraw
	go drawEncoder(ctx, encoder1, 25, 1, 60*time.Millisecond, playTypePercent)

	encoder2, err := encoder.New(
		encoder.CellOpts(cell.FgColor(cell.ColorGreen)),
		encoder.Label("E2", cell.FgColor(cell.ColorGreen)),
		encoder.OscRoute("/remote/enc/2", oscAddr, oscPort),
	)
	//log.Fatal(fmt.Sprintf("%+v", encoder3))
	if err != nil {
		panic(err)
	}
	// TODO: rename redraw
	go drawEncoder(ctx, encoder2, 25, 1, 60*time.Millisecond, playTypePercent)

	encoder3, err := encoder.New(
		encoder.CellOpts(cell.FgColor(cell.ColorGreen)),
		encoder.Label("E3", cell.FgColor(cell.ColorGreen)),
		encoder.OscRoute("/remote/enc/3", oscAddr, oscPort),
	)
	if err != nil {
		panic(err)
	}

	go drawEncoder(ctx, encoder3, 25, 1, 60*time.Millisecond, playTypePercent)

	display, err := segmentdisplay.New()
	if err != nil {
		panic(err)
	}

	keyStates := map[int]int{
		1: 0,
		2: 0,
		3: 0,
	}
	// TODO: button release requires fast double clicks
	// this should send 1 on press and 0 on relese
	k1, err := button.New("K1", func() error {
		keyStates[1] = 1 - keyStates[1]
		client := osc.NewClient(oscAddr, oscPort)
		msg := osc.NewMessage("/remote/key/1")
		msg.Append(int32(keyStates[1]))
		client.Send(msg)

		return display.Write([]*segmentdisplay.TextChunk{
			segmentdisplay.NewChunk(fmt.Sprintf("%d", keyStates[1])),
		})
	},
		button.GlobalKey('1'),
		button.WidthFor("K1"),
	)

	if err != nil {
		panic(err)
	}

	k2, err := button.New("K2", func() error {
		keyStates[2] = 1 - keyStates[2]
		client := osc.NewClient(oscAddr, oscPort)
		msg := osc.NewMessage("/remote/key/2")
		msg.Append(int32(keyStates[2]))
		client.Send(msg)

		return display.Write([]*segmentdisplay.TextChunk{
			segmentdisplay.NewChunk(fmt.Sprintf("%d", keyStates[2])),
		})
	},
		button.GlobalKey('2'),
		button.WidthFor("K2"),
	)

	if err != nil {
		panic(err)
	}

	k3, err := button.New("K3", func() error {
		keyStates[3] = 1 - keyStates[3]
		client := osc.NewClient(oscAddr, oscPort)
		msg := osc.NewMessage("/remote/key/3")
		msg.Append(int32(keyStates[3]))
		client.Send(msg)

		return display.Write([]*segmentdisplay.TextChunk{
			segmentdisplay.NewChunk(fmt.Sprintf("%d", keyStates[3])),
		})
	},
		button.GlobalKey('3'),
		button.WidthFor("K3"),
	)

	if err != nil {
		panic(err)
	}

	nornsScreen, err := text.New()
	if err != nil {
		panic(err)
	}
	go drawScreen(ctx, nornsScreen, 100*time.Millisecond)

	c, err := container.New(
		t,
		container.Border(linestyle.Light),
		container.BorderTitle("PRESS Q TO QUIT"),
		container.SplitHorizontal(
			container.Top(
				container.PlaceWidget(nornsScreen),
				container.AlignHorizontal(align.HorizontalCenter),
				container.AlignVertical(align.VerticalMiddle),
			),
			container.Bottom(

				container.SplitVertical(
					container.Left(
						container.SplitHorizontal(
							container.Top(container.PlaceWidget(encoder1)),
							container.Bottom(container.PlaceWidget(k1)),
						),
					),
					container.Right(
						container.SplitVertical(
							container.Left(
								container.SplitHorizontal(
									container.Top(container.PlaceWidget(encoder2)),
									container.Bottom(container.PlaceWidget(k2)),
								),
							),
							container.Right(
								container.SplitHorizontal(
									container.Top(container.PlaceWidget(encoder3)),
									container.Bottom(container.PlaceWidget(k3)),
								),
							),
						),
					),
					container.SplitPercent(33),
				),
			),
		),

	)
	if err != nil {
		panic(err)
	}

	quitter := func(k *terminalapi.Keyboard) {
		if k.Key == 'q' || k.Key == 'Q' {
			cancel()
		}
	}

	if err := termdash.Run(ctx, t, c, termdash.KeyboardSubscriber(quitter), termdash.RedrawInterval(60*time.Second)); err != nil {
		panic(err)
	}
}

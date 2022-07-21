package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/hypebeast/go-osc/osc"
	"github.com/mum4k/termdash"
	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/container"
	"github.com/mum4k/termdash/linestyle"
	"github.com/mum4k/termdash/terminal/tcell"
	"github.com/mum4k/termdash/terminal/terminalapi"
	"github.com/mum4k/termdash/widgets/button"
	"github.com/mum4k/termdash/widgets/segmentdisplay"
	"github.com/zzsnzmn/osctl/internal/encoder"
)

// drawEncoder continuously changes the displayed percent value on the encoder by the
// step once every delay. Exits when the context expires.
func drawEncoder(ctx context.Context, d *encoder.Encoder, start, step int, delay time.Duration) {
	progress := start
	mult := 1

	ticker := time.NewTicker(delay)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:

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
			if err := d.Percent(progress); err != nil {
				panic(err)
			}

		case <-ctx.Done():
			return
		}
	}
}

func enc(oscAddr string, oscRoute string, oscPort int, encoderLabel string) *encoder.Encoder {
	e, err := encoder.New(
		encoder.CellOpts(cell.FgColor(cell.ColorGreen)),
		encoder.Label(encoderLabel, cell.FgColor(cell.ColorGreen)),
		encoder.HideTextProgress(),
		encoder.OscRoute(oscRoute, oscAddr, oscPort),
	)
	if err != nil {
		panic(err)
	}
	return e
}

// btn creates a closure to track button press states and returns a callback function for use with the Button widget.
func btn(oscAddr string, oscRoute string, oscPort int, encoderLabel string, display *segmentdisplay.SegmentDisplay) func() error {
	keyState := 0
	client := osc.NewClient(oscAddr, oscPort)
	return func() error {
		keyState = 1 - keyState
		msg := osc.NewMessage(oscRoute)
		msg.Append(int32(keyState))
		err := client.Send(msg)
		if err != nil {
			log.Printf("error sending osc message: %+v", msg)
		}
		return display.Write([]*segmentdisplay.TextChunk{
			segmentdisplay.NewChunk(fmt.Sprintf("%d", keyState)),
		})
	}
}

// newGui returns a container with an even 33% vertical split for each of the encoders and buttons provided.
func newGui(t *tcell.Terminal, e1, e2, e3 *encoder.Encoder, k1, k2, k3 *button.Button) (*container.Container, error) {
	return container.New(
		t,
		container.Border(linestyle.Light),
		container.BorderTitle("PRESS Q TO QUIT"),

		container.SplitVertical(
			container.Left(
				container.SplitHorizontal(
					container.Top(container.PlaceWidget(e1)),
					container.Bottom(container.PlaceWidget(k1)),
				),
			),
			container.Right(
				container.SplitVertical(
					container.Left(
						container.SplitHorizontal(
							container.Top(container.PlaceWidget(e2)),
							container.Bottom(container.PlaceWidget(k2)),
						),
					),
					container.Right(
						container.SplitHorizontal(
							container.Top(container.PlaceWidget(e3)),
							container.Bottom(container.PlaceWidget(k3)),
						),
					),
				),
			),
			container.SplitPercent(33),
		),
	)
}

func main() {

	// set up flags
	// TODO: read arg here to set range
	// TODO: read arg here to set osc msg type
	// TODO: read arg here to set osc msg route
	oscAddrFlag := flag.String("addr", "127.0.0.1", "the ip or hostname to send OSC messages to")
	oscPortFlag := flag.Int("port", 10111, "the port to send OSC messages to")
	flag.Parse()

	t, err := tcell.New()
	if err != nil {
		panic(err)
	}
	defer t.Close()

	oscAddr := *oscAddrFlag
	oscPort := *oscPortFlag
	ctx, cancel := context.WithCancel(context.Background())
	e1 := enc(oscAddr, "/remote/enc/1", oscPort, "E1")
	e2 := enc(oscAddr, "/remote/enc/2", oscPort, "E2")
	e3 := enc(oscAddr, "/remote/enc/3", oscPort, "E3")

	// TODO: this is kinda messy, but handles callbacks for drawing the encoder
	go drawEncoder(ctx, e1, 25, 1, 60*time.Millisecond)
	go drawEncoder(ctx, e2, 25, 1, 60*time.Millisecond)
	go drawEncoder(ctx, e3, 25, 1, 60*time.Millisecond)

	display, err := segmentdisplay.New()
	if err != nil {
		panic(err)
	}

	// TODO: button release requires fast double clicks
	// this should send 1 on press and 0 on release, but the way that mouse clicks with with the termGUI it's
	k1, _ := button.New("K1", btn(oscAddr, "/remote/key/1", oscPort, "K1", display))
	k2, _ := button.New("K2", btn(oscAddr, "/remote/key/2", oscPort, "K2", display))
	k3, _ := button.New("K3", btn(oscAddr, "/remote/key/3", oscPort, "K2", display))

	c, err := newGui(t, e1, e2, e3, k1, k2, k3)
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

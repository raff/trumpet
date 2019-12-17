package main

import (
	"image/color"
	"log"
	"strconv"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/speaker"

	"gioui.org/app"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"

	"gioui.org/font/gofont"
	"gioui.org/widget/material"

	"golang.org/x/exp/shiny/materialdesign/icons"
)

type SelectStreamer struct {
	notes   map[string]beep.StreamSeeker
	stream  beep.StreamSeeker
	silence [][2]float64
	start   int
}

func SelectStream(notes map[string]beep.StreamSeeker, format beep.Format, start time.Duration) *SelectStreamer {
	return &SelectStreamer{
		notes:   notes,
		silence: make([][2]float64, format.SampleRate.N(time.Second/2)),
		start:   format.SampleRate.N(start),
	}
}

func (s *SelectStreamer) Select(k string) {
	playing := s.stream != nil

	if st, ok := s.notes[k]; ok {
		//log.Println("select", k)
		s.stream = st

		if playing {
			s.stream.Seek(s.start)
		} else {
			s.stream.Seek(0)
		}
	} else {
		//log.Println("unselect", k)
		s.stream = nil
	}
}

func (s *SelectStreamer) Stream(samples [][2]float64) (n int, ok bool) {
	if s.stream == nil {
		return copy(samples, s.silence), true
	}

	n, ok = s.stream.Stream(samples)
	if n == 0 {
		s.stream = nil
	}

	return
}

func (s *SelectStreamer) Err() error {
	if s.stream != nil {
		return s.stream.Err()
	}

	return nil
}

var (
	// The key is {harmonics}{valves}
	// where {harmonics} is 0-5 and {valves} is 0 to 123
	//
	// i.e. C fundamental is 00 and D on the 1st harmonics is 113
	//
	// note that the note names are in concert pitch so we need to transpose

	notes = map[string]beep.StreamSeeker{
		// fundamental (C4)
		"0123": &EmbeddedStream{buf: notes_audio["%1-E 3"]}, // F#3
		"013":  &EmbeddedStream{buf: notes_audio["%2-F 3"]}, // G3
		"023":  &EmbeddedStream{buf: notes_audio["%3-F+3"]}, // G#3
		"012":  &EmbeddedStream{buf: notes_audio["%4-G 3"]}, // A3
		"01":   &EmbeddedStream{buf: notes_audio["%5-G+3"]}, // Bb3
		"02":   &EmbeddedStream{buf: notes_audio["%6-A 3"]}, // B3
		"00":   &EmbeddedStream{buf: notes_audio["%7-A+3"]}, // C4

		// 1st harmonics (G4)
		"1123": &EmbeddedStream{buf: notes_audio["%8-B 3"]}, // C#4
		"113":  &EmbeddedStream{buf: notes_audio["01-C 4"]}, // D4
		"123":  &EmbeddedStream{buf: notes_audio["02-C+4"]}, // Eb4
		"112":  &EmbeddedStream{buf: notes_audio["03-D 4"]}, // E4
		"11":   &EmbeddedStream{buf: notes_audio["04-D+4"]}, // F4
		"12":   &EmbeddedStream{buf: notes_audio["05-E 4"]}, // F#4
		"10":   &EmbeddedStream{buf: notes_audio["06-F 4"]}, // G4

		// 2nd harmonics (C5)
		"223": &EmbeddedStream{buf: notes_audio["07-F+4"]}, // G#5
		"212": &EmbeddedStream{buf: notes_audio["08-G 4"]}, // A5
		"21":  &EmbeddedStream{buf: notes_audio["09-G+4"]}, // Bb5
		"22":  &EmbeddedStream{buf: notes_audio["10-A 4"]}, // B5
		"20":  &EmbeddedStream{buf: notes_audio["11-A+4"]}, // C5

		// 3rd harmonics (E5)
		"312": &EmbeddedStream{buf: notes_audio["12-B 4"]}, // C#4
		"31":  &EmbeddedStream{buf: notes_audio["13-C 5"]}, // D5
		"32":  &EmbeddedStream{buf: notes_audio["14-C+5"]}, // Eb5
		"30":  &EmbeddedStream{buf: notes_audio["15-D 5"]}, // E5

		// 4th harmonics (G5)
		"41": &EmbeddedStream{buf: notes_audio["16-D+5"]}, // F5
		"42": &EmbeddedStream{buf: notes_audio["17-E 5"]}, // F#5
		"40": &EmbeddedStream{buf: notes_audio["18-F 5"]}, // G5

		// 5th harmonics (Bb5)
		"51": &EmbeddedStream{buf: notes_audio["19-F+5"]}, // G#5
		"52": &EmbeddedStream{buf: notes_audio["20-G 5"]}, // A5
		"50": &EmbeddedStream{buf: notes_audio["21-G+5"]}, // Bb5

		// 6th harmonics (C6)
		"62": &EmbeddedStream{buf: notes_audio["22-A 5"]}, // B5
		"60": &EmbeddedStream{buf: notes_audio["23-A+5"]}, // C6

		// 7th harmonics (D6)
		"72": &EmbeddedStream{buf: notes_audio["24-B 5"]}, // C#6
		"70": &EmbeddedStream{buf: notes_audio["25-C 6"]}, // D6

		// 8th harmonics (E6)
		"82": &EmbeddedStream{buf: notes_audio["26-C+6"]}, // Eb6
		"80": &EmbeddedStream{buf: notes_audio["27-D 6"]}, // E6

		// 9th harmonics (F6)
		"90": &EmbeddedStream{buf: notes_audio["28-D+6"]}, // F6
	}

	title material.Label

	blist = layout.List{Axis: layout.Vertical}

	harmonics = [10]widget.Button{}

	iconValves [3]*material.Icon
	valves     = []*widget.Button{
		new(widget.Button),
		new(widget.Button),
		new(widget.Button)}

	hnames = []string{
		"F6",
		"E6",
		"D6",
		"C6",
		"Bb5",
		"G5",
		"E5",
		"C5",
		"G4",
		"C4",
	}
)

func render(gtx *layout.Context, theme *material.Theme, kk map[string]int) {
	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func() {
			title.Layout(gtx)
		}),

		layout.Rigid(func() {
			layout.Flex{Axis: layout.Horizontal, Alignment: layout.End}.Layout(gtx,
				layout.Rigid(func() {
					hpress := -1

					blist.Layout(gtx, 10, func(i int) {
						h := 9 - i

						if harmonics[h].Pressed(gtx) {
							hpress = h
						}

						layout.UniformInset(unit.Dp(5)).Layout(gtx, func() {
							gtx.Constraints.Width.Min = 200
							gtx.Constraints.Height.Min = 60

							theme.Button(hnames[i]).Layout(gtx, &harmonics[h])
						})
					})

					if hpress >= 0 {
						kk["h"] = hpress
					} else {
						delete(kk, "h")
					}
				}),

				layout.Rigid(func() {
					layout.UniformInset(unit.Dp(30)).Layout(gtx, func() {})
				}),

				layout.Rigid(func() {
					layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func() {
							layout.UniformInset(unit.Dp(5)).Layout(gtx, func() {

								if valves[0].Pressed(gtx) {
									kk["1"] = 1
								} else {
									delete(kk, "1")
								}
								theme.IconButton(iconValves[2]).Layout(gtx, valves[2])
							})
						}),
						layout.Rigid(func() {
							layout.UniformInset(unit.Dp(5)).Layout(gtx, func() {
								if valves[1].Pressed(gtx) {
									kk["2"] = 2
								} else {
									delete(kk, "2")
								}
								theme.IconButton(iconValves[1]).Layout(gtx, valves[1])
							})
						}),
						layout.Rigid(func() {
							layout.UniformInset(unit.Dp(5)).Layout(gtx, func() {
								if valves[2].Pressed(gtx) {
									kk["3"] = 3
								} else {
									delete(kk, "3")
								}
								theme.IconButton(iconValves[0]).Layout(gtx, valves[0])
							})
						}),
					)
				}))
		}))
}

func main() {
	gofont.Register()

	if ic, err := material.NewIcon(icons.ImageLooksOne); err != nil {
		log.Fatal(err)
	} else {
		iconValves[0] = ic
	}

	if ic, err := material.NewIcon(icons.ImageLooksTwo); err != nil {
		log.Fatal(err)
	} else {
		iconValves[1] = ic
	}

	if ic, err := material.NewIcon(icons.ImageLooks3); err != nil {
		log.Fatal(err)
	} else {
		iconValves[2] = ic
	}

	speaker.Init(notes_format.SampleRate, notes_format.SampleRate.N(time.Second/30))
	stream := SelectStream(notes, notes_format, time.Second/2)
	resampler := beep.ResampleRatio(4, 1, stream)
	volume := &effects.Volume{Streamer: resampler, Base: 2}
	go speaker.Play(volume)

	go func() {
		theme := material.NewTheme()

		title = theme.H3("Trumpet simulator")
		title.Color = color.RGBA{127, 0, 0, 255}
		title.Alignment = text.Middle

		w := app.NewWindow(
			app.Size(unit.Dp(250), unit.Dp(600)),
			app.Title("Trumpet Simulator"))
		gtx := layout.NewContext(w.Queue())

		kk := map[string]int{}

		processKeys := func() string {
			h, ok := kk["h"]
			if !ok {
				return ""
			}

			n := 0

			if v := kk["1"]; v > 0 {
				n = n*10 + v
			}
			if v := kk["2"]; v > 0 {
				n = n*10 + v
			}
			if v := kk["3"]; v > 0 {
				n = n*10 + v
			}

			//log.Printf("processKeys %v %v", h, n)
			return strconv.Itoa(h) + strconv.Itoa(n)
		}

		prev := ""

		for {
			e := <-w.Events()
			switch e := e.(type) {
			case system.DestroyEvent:
				if e.Err != nil {
					log.Println(e.Err)
				}
				return
			case system.FrameEvent:
				gtx.Reset(e.Config, e.Size)
				render(gtx, theme, kk)
				e.Frame(gtx.Ops)

				k := processKeys()
				if k != prev {
					prev = k

					speaker.Lock()
					stream.Select(k)
					speaker.Unlock()
				}

			case key.Event:
				switch e.Name {
				case "`", "1", "2", "3", "4", "5", "6", "7", "8", "9":
					n, _ := strconv.Atoi(e.Name)
					kk["h"] = n

				case "0":
					kk["1"] = 1
				case "-":
					kk["2"] = 2
				case "=":
					kk["3"] = 3

				case "[":
					speaker.Lock()
					volume.Volume -= 0.1
					speaker.Unlock()

				case "]":
					speaker.Lock()
					volume.Volume += 0.1
					speaker.Unlock()

				default:
					break
				}

				k := processKeys()
				if k != prev {
					prev = k

					speaker.Lock()
					stream.Select(k)
					speaker.Unlock()
				}

			case key.UpEvent:
				switch e.Name {
				case "`", "1", "2", "3", "4", "5", "6", "7", "8", "9":
					delete(kk, "h")
				case "0":
					delete(kk, "1")
				case "-":
					delete(kk, "2")
				case "=":
					delete(kk, "3")
				default:
					break
				}

				k := processKeys()
				if k != prev {
					prev = k
					speaker.Lock()
					stream.Select(k)
					speaker.Unlock()
				}
			}
		}
	}()

	app.Main()
}

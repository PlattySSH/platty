# Sol, Platty's mascot

A wavy white body inside a light-grey glass ring, with two glass eyes. Sol is drawn with
vectors in `internal/ui/mascot`, and everything in this folder is exported from there:

```sh
go run ./cmd/platty-mascot -o assets/mascot
```

| File | What it is |
| --- | --- |
| `sol-<n>.png` | Sol on a transparent background (16–1024 px) |
| `sol-light-<n>.png` | White glass with a dark rim, for light backgrounds |
| `sol-happy-<n>.png` | Smiling eyes (^ ^) |
| `sol-dark-<n>.png` | On Platty's dark background |
| `sol.gif` | Looping animation (4.2 s, 25 fps): the waves flow, Sol blinks and looks around |
| `frames/sol-NNN.png` | The 105 animation frames, with transparency |
| `platty.ico` | Windows icon (16–256 px) |
| `github/platty-github-*.png` | GitHub organization avatar (1024 px): black, graphite and pure |

The executable's icon is built in `cmd/platty/winres` with
`go run github.com/tc-hib/go-winres@v0.3.3 make --arch amd64` (run from `cmd/platty`).

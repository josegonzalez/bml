# bml

A Go parser for BML (Binary Markup Language) files used by the [ares emulator](https://ares-emu.net/).

## Installation

```bash
go get github.com/josegonzalez/bml
```

## Usage

### Struct Tags

```go
type Settings struct {
    Video struct {
        Driver     string  `bml:"Driver"`
        Multiplier int     `bml:"Multiplier"`
        Luminance  float64 `bml:"Luminance"`
    } `bml:"Video"`
}

var s Settings
bml.Unmarshal(data, &s)
```

### Node API

```go
doc, _ := bml.Parse(data)

// Read values
driver := doc.Root.Get("Video/Driver").String("")
mult := doc.Root.Get("Video/Multiplier").Int(1)

// Modify values
doc.Root.Set("Video/Driver", "OpenGL")
doc.Root.Get("Video").SetInt("Multiplier", 3)

// Serialize back
output := bml.Serialize(doc)
```

## BML Format

```text
Video
  Driver: Metal
  Multiplier: 2
Audio
  Driver: SDL
  Volume: 1.0
```

## License

MIT

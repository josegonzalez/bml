package bml

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

// === Parser Tests ===

func TestParseEmpty(t *testing.T) {
	doc, err := Parse([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.Root == nil {
		t.Fatal("expected non-nil root")
	}
	if len(doc.Root.Children) != 0 {
		t.Errorf("expected 0 children, got %d", len(doc.Root.Children))
	}
}

func TestParseSingleNode(t *testing.T) {
	doc, err := Parse([]byte("Video"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(doc.Root.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(doc.Root.Children))
	}
	if doc.Root.Children[0].Name != "Video" {
		t.Errorf("expected name 'Video', got %q", doc.Root.Children[0].Name)
	}
	if doc.Root.Children[0].Value != "" {
		t.Errorf("expected empty value, got %q", doc.Root.Children[0].Value)
	}
}

func TestParseNodeWithColonValue(t *testing.T) {
	doc, err := Parse([]byte("Driver: Metal"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(doc.Root.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(doc.Root.Children))
	}
	if doc.Root.Children[0].Value != "Metal" {
		t.Errorf("expected value 'Metal', got %q", doc.Root.Children[0].Value)
	}
}

func TestParseNodeWithEqualsValue(t *testing.T) {
	doc, err := Parse([]byte("Driver=Metal"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.Root.Children[0].Value != "Metal" {
		t.Errorf("expected value 'Metal', got %q", doc.Root.Children[0].Value)
	}
}

func TestParseNodeWithQuotedValue(t *testing.T) {
	doc, err := Parse([]byte(`Driver="Metal GPU"`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.Root.Children[0].Value != "Metal GPU" {
		t.Errorf("expected value 'Metal GPU', got %q", doc.Root.Children[0].Value)
	}
}

func TestParseNestedNodes(t *testing.T) {
	input := `Video
  Driver: Metal
  Multiplier: 2
Audio
  Driver: SDL`

	doc, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Root.Children) != 2 {
		t.Fatalf("expected 2 root children, got %d", len(doc.Root.Children))
	}

	video := doc.Root.Children[0]
	if video.Name != "Video" {
		t.Errorf("expected 'Video', got %q", video.Name)
	}
	if len(video.Children) != 2 {
		t.Fatalf("expected 2 video children, got %d", len(video.Children))
	}
	if video.Children[0].Name != "Driver" || video.Children[0].Value != "Metal" {
		t.Errorf("unexpected video child: %+v", video.Children[0])
	}

	audio := doc.Root.Children[1]
	if audio.Name != "Audio" {
		t.Errorf("expected 'Audio', got %q", audio.Name)
	}
}

func TestParseDeeplyNested(t *testing.T) {
	input := `Paths
  SuperFamicom
    GameBoy
      Path: /games`

	doc, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	node := doc.Root.Get("Paths/SuperFamicom/GameBoy/Path")
	if node == nil {
		t.Fatal("expected to find deeply nested node")
	}
	if node.Value != "/games" {
		t.Errorf("expected '/games', got %q", node.Value)
	}
}

func TestParseComments(t *testing.T) {
	input := `// This is a comment
Video
  // Another comment
  Driver: Metal`

	doc, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Root.Children) != 1 {
		t.Fatalf("expected 1 child (comments skipped), got %d", len(doc.Root.Children))
	}
	if doc.Root.Children[0].Name != "Video" {
		t.Errorf("expected 'Video', got %q", doc.Root.Children[0].Name)
	}
}

func TestParseInlineComment(t *testing.T) {
	input := `Driver: Metal // This is a comment`

	doc, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if doc.Root.Children[0].Value != "Metal" {
		t.Errorf("expected 'Metal', got %q", doc.Root.Children[0].Value)
	}
}

func TestParseEmptyLines(t *testing.T) {
	input := `Video

  Driver: Metal

Audio`

	doc, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Root.Children) != 2 {
		t.Fatalf("expected 2 children (empty lines skipped), got %d", len(doc.Root.Children))
	}
}

func TestParseWindowsLineEndings(t *testing.T) {
	input := "Video\r\n  Driver: Metal\r\n"

	doc, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Root.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(doc.Root.Children))
	}
}

func TestParseOldMacLineEndings(t *testing.T) {
	input := "Video\r  Driver: Metal\r"

	doc, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Root.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(doc.Root.Children))
	}
}

func TestParseTabIndentation(t *testing.T) {
	input := "Video\n\tDriver: Metal"

	doc, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if doc.Root.Get("Video/Driver") == nil {
		t.Error("expected to find Video/Driver with tab indentation")
	}
}

func TestParseUnclosedQuote(t *testing.T) {
	input := `Driver="Metal`

	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for unclosed quote")
	}
	if !strings.Contains(err.Error(), "unclosed quote") {
		t.Errorf("expected 'unclosed quote' error, got: %v", err)
	}
}

func TestParseInvalidNodeName(t *testing.T) {
	input := `  : value`

	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for invalid node name")
	}
}

func TestParseMultilineValue(t *testing.T) {
	input := `Description
  : Line 1
  : Line 2
  : Line 3`

	doc, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	node := doc.Root.Children[0]
	expected := "Line 1\nLine 2\nLine 3"
	if node.Value != expected {
		t.Errorf("expected %q, got %q", expected, node.Value)
	}
}

func TestParseAttributes(t *testing.T) {
	input := `Node attr1=value1 attr2: value2`

	doc, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	node := doc.Root.Children[0]
	if len(node.Children) != 2 {
		t.Fatalf("expected 2 attributes, got %d", len(node.Children))
	}
	if node.Children[0].Name != "attr1" || node.Children[0].Value != "value1" {
		t.Errorf("unexpected attr1: %+v", node.Children[0])
	}
	if node.Children[1].Name != "attr2" || node.Children[1].Value != "value2" {
		t.Errorf("unexpected attr2: %+v", node.Children[1])
	}
}

func TestParseAttributeWithInlineComment(t *testing.T) {
	input := `Node attr1=value1 // comment`

	doc, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	node := doc.Root.Children[0]
	if len(node.Children) != 1 {
		t.Fatalf("expected 1 attribute, got %d", len(node.Children))
	}
}

func TestParseEmptyEqualsValue(t *testing.T) {
	input := "Node="

	doc, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if doc.Root.Children[0].Value != "" {
		t.Errorf("expected empty value, got %q", doc.Root.Children[0].Value)
	}
}

func TestParseColonNoSpace(t *testing.T) {
	input := "Node:value"

	doc, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if doc.Root.Children[0].Value != "value" {
		t.Errorf("expected 'value', got %q", doc.Root.Children[0].Value)
	}
}

func TestParseValidNameChars(t *testing.T) {
	input := "Node-Name.123: value"

	doc, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if doc.Root.Children[0].Name != "Node-Name.123" {
		t.Errorf("expected 'Node-Name.123', got %q", doc.Root.Children[0].Name)
	}
}

// === Node Accessor Tests ===

func TestNodeGetValid(t *testing.T) {
	doc, _ := Parse([]byte("Video\n  Driver: Metal"))

	node := doc.Root.Get("Video/Driver")
	if node == nil {
		t.Fatal("expected to find node")
	}
	if node.Value != "Metal" {
		t.Errorf("expected 'Metal', got %q", node.Value)
	}
}

func TestNodeGetInvalid(t *testing.T) {
	doc, _ := Parse([]byte("Video\n  Driver: Metal"))

	node := doc.Root.Get("Audio/Driver")
	if node != nil {
		t.Error("expected nil for non-existent path")
	}
}

func TestNodeGetEmptyPath(t *testing.T) {
	doc, _ := Parse([]byte("Video"))

	node := doc.Root.Get("")
	if node != doc.Root {
		t.Error("expected root for empty path")
	}
}

func TestNodeGetNil(t *testing.T) {
	var node *Node
	result := node.Get("path")
	if result != nil {
		t.Error("expected nil for nil node")
	}
}

func TestNodeString(t *testing.T) {
	doc, _ := Parse([]byte("Driver: Metal"))

	val := doc.Root.Get("Driver").String("default")
	if val != "Metal" {
		t.Errorf("expected 'Metal', got %q", val)
	}
}

func TestNodeStringFallback(t *testing.T) {
	doc, _ := Parse([]byte("Video"))

	val := doc.Root.Get("Audio").String("default")
	if val != "default" {
		t.Errorf("expected 'default', got %q", val)
	}
}

func TestNodeStringNil(t *testing.T) {
	var node *Node
	val := node.String("default")
	if val != "default" {
		t.Errorf("expected 'default', got %q", val)
	}
}

func TestNodeBoolTrue(t *testing.T) {
	doc, _ := Parse([]byte("Enabled: true"))

	val := doc.Root.Get("Enabled").Bool(false)
	if val != true {
		t.Error("expected true")
	}
}

func TestNodeBoolFalse(t *testing.T) {
	doc, _ := Parse([]byte("Enabled: false"))

	val := doc.Root.Get("Enabled").Bool(true)
	if val != false {
		t.Error("expected false")
	}
}

func TestNodeBoolInvalid(t *testing.T) {
	doc, _ := Parse([]byte("Enabled: yes"))

	val := doc.Root.Get("Enabled").Bool(true)
	if val != true {
		t.Error("expected fallback value true")
	}
}

func TestNodeBoolNil(t *testing.T) {
	var node *Node
	val := node.Bool(true)
	if val != true {
		t.Error("expected fallback value true")
	}
}

func TestNodeInt(t *testing.T) {
	doc, _ := Parse([]byte("Count: 42"))

	val := doc.Root.Get("Count").Int(0)
	if val != 42 {
		t.Errorf("expected 42, got %d", val)
	}
}

func TestNodeIntInvalid(t *testing.T) {
	doc, _ := Parse([]byte("Count: abc"))

	val := doc.Root.Get("Count").Int(99)
	if val != 99 {
		t.Errorf("expected fallback 99, got %d", val)
	}
}

func TestNodeIntNil(t *testing.T) {
	var node *Node
	val := node.Int(99)
	if val != 99 {
		t.Errorf("expected fallback 99, got %d", val)
	}
}

func TestNodeFloat(t *testing.T) {
	doc, _ := Parse([]byte("Value: 3.14"))

	val := doc.Root.Get("Value").Float(0)
	if val != 3.14 {
		t.Errorf("expected 3.14, got %f", val)
	}
}

func TestNodeFloatInvalid(t *testing.T) {
	doc, _ := Parse([]byte("Value: abc"))

	val := doc.Root.Get("Value").Float(1.5)
	if val != 1.5 {
		t.Errorf("expected fallback 1.5, got %f", val)
	}
}

func TestNodeFloatNil(t *testing.T) {
	var node *Node
	val := node.Float(1.5)
	if val != 1.5 {
		t.Errorf("expected fallback 1.5, got %f", val)
	}
}

// === Node Mutation Tests ===

func TestNodeSet(t *testing.T) {
	doc, _ := Parse([]byte("Video"))

	doc.Root.Get("Video").Set("Driver", "Metal")
	node := doc.Root.Get("Video/Driver")
	if node == nil {
		t.Fatal("expected node to be created")
	}
	if node.Value != "Metal" {
		t.Errorf("expected 'Metal', got %q", node.Value)
	}
}

func TestNodeSetUpdate(t *testing.T) {
	doc, _ := Parse([]byte("Video\n  Driver: OpenGL"))

	doc.Root.Get("Video").Set("Driver", "Metal")
	node := doc.Root.Get("Video/Driver")
	if node.Value != "Metal" {
		t.Errorf("expected 'Metal', got %q", node.Value)
	}
}

func TestNodeSetNestedCreate(t *testing.T) {
	doc, _ := Parse([]byte(""))
	doc.Root.Set("Video/Driver", "Metal")

	node := doc.Root.Get("Video/Driver")
	if node == nil {
		t.Fatal("expected nested node to be created")
	}
	if node.Value != "Metal" {
		t.Errorf("expected 'Metal', got %q", node.Value)
	}
}

func TestNodeSetNil(t *testing.T) {
	var node *Node
	result := node.Set("path", "value")
	if result != nil {
		t.Error("expected nil for nil node")
	}
}

func TestNodeSetBool(t *testing.T) {
	doc, _ := Parse([]byte(""))
	doc.Root.SetBool("Enabled", true)

	if doc.Root.Get("Enabled").Value != "true" {
		t.Error("expected 'true'")
	}

	doc.Root.SetBool("Disabled", false)
	if doc.Root.Get("Disabled").Value != "false" {
		t.Error("expected 'false'")
	}
}

func TestNodeSetInt(t *testing.T) {
	doc, _ := Parse([]byte(""))
	doc.Root.SetInt("Count", 42)

	if doc.Root.Get("Count").Value != "42" {
		t.Error("expected '42'")
	}
}

func TestNodeSetFloat(t *testing.T) {
	doc, _ := Parse([]byte(""))
	doc.Root.SetFloat("Value", 3.14)

	val := doc.Root.Get("Value").Float(0)
	if val != 3.14 {
		t.Errorf("expected 3.14, got %f", val)
	}
}

func TestNodeRemove(t *testing.T) {
	doc, _ := Parse([]byte("Video\n  Driver: Metal\n  Count: 2"))

	removed := doc.Root.Get("Video").Remove("Driver")
	if !removed {
		t.Error("expected Remove to return true")
	}

	if doc.Root.Get("Video/Driver") != nil {
		t.Error("expected node to be removed")
	}

	// Count should still exist
	if doc.Root.Get("Video/Count") == nil {
		t.Error("expected Count to still exist")
	}
}

func TestNodeRemoveNonExistent(t *testing.T) {
	doc, _ := Parse([]byte("Video"))

	removed := doc.Root.Get("Video").Remove("Driver")
	if removed {
		t.Error("expected Remove to return false for non-existent node")
	}
}

func TestNodeRemoveNestedPath(t *testing.T) {
	doc, _ := Parse([]byte("Video\n  Settings\n    Driver: Metal"))

	removed := doc.Root.Remove("Video/Settings")
	if !removed {
		t.Error("expected Remove to return true")
	}

	if doc.Root.Get("Video/Settings") != nil {
		t.Error("expected nested node to be removed")
	}
}

func TestNodeRemoveNil(t *testing.T) {
	var node *Node
	removed := node.Remove("path")
	if removed {
		t.Error("expected false for nil node")
	}
}

func TestNodeRemoveEmptyPath(t *testing.T) {
	doc, _ := Parse([]byte("Video"))
	removed := doc.Root.Remove("")
	if removed {
		t.Error("expected false for empty path")
	}
}

func TestNodeRemoveInvalidPath(t *testing.T) {
	doc, _ := Parse([]byte("Video"))
	removed := doc.Root.Remove("Audio/Driver")
	if removed {
		t.Error("expected false for invalid path")
	}
}

// === Serialization Tests ===

func TestSerializeEmpty(t *testing.T) {
	doc := &Document{Root: &Node{}}
	data := Serialize(doc)
	if len(data) != 0 {
		t.Errorf("expected empty output, got %q", string(data))
	}
}

func TestSerializeNil(t *testing.T) {
	data := Serialize(nil)
	if data != nil {
		t.Error("expected nil for nil document")
	}

	data = Serialize(&Document{})
	if data != nil {
		t.Error("expected nil for nil root")
	}
}

func TestSerializeSingleNode(t *testing.T) {
	doc := &Document{Root: &Node{
		Children: []*Node{{Name: "Video"}},
	}}
	data := Serialize(doc)
	expected := "Video\n"
	if string(data) != expected {
		t.Errorf("expected %q, got %q", expected, string(data))
	}
}

func TestSerializeNodeWithValue(t *testing.T) {
	doc := &Document{Root: &Node{
		Children: []*Node{{Name: "Driver", Value: "Metal"}},
	}}
	data := Serialize(doc)
	expected := "Driver: Metal\n"
	if string(data) != expected {
		t.Errorf("expected %q, got %q", expected, string(data))
	}
}

func TestSerializeNested(t *testing.T) {
	doc := &Document{Root: &Node{
		Children: []*Node{
			{
				Name: "Video",
				Children: []*Node{
					{Name: "Driver", Value: "Metal"},
				},
			},
		},
	}}
	data := Serialize(doc)
	expected := "Video\n  Driver: Metal\n"
	if string(data) != expected {
		t.Errorf("expected %q, got %q", expected, string(data))
	}
}

func TestSerializeMultilineValue(t *testing.T) {
	doc := &Document{Root: &Node{
		Children: []*Node{{Name: "Desc", Value: "Line1\nLine2"}},
	}}
	data := Serialize(doc)
	expected := "Desc\n  : Line1\n  : Line2\n"
	if string(data) != expected {
		t.Errorf("expected %q, got %q", expected, string(data))
	}
}

func TestSerializeRoundTrip(t *testing.T) {
	input := `Video
  Driver: Metal
  Multiplier: 2
Audio
  Driver: SDL
  Volume: 1.0
`
	doc, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	output := Serialize(doc)
	doc2, err := Parse(output)
	if err != nil {
		t.Fatalf("re-parse error: %v", err)
	}

	// Verify key values
	if doc2.Root.Get("Video/Driver").String("") != "Metal" {
		t.Error("Video/Driver mismatch after round-trip")
	}
	if doc2.Root.Get("Audio/Volume").Float(0) != 1.0 {
		t.Error("Audio/Volume mismatch after round-trip")
	}
}

// === Marshal/Unmarshal Tests ===

type TestVideoSettings struct {
	Driver     string  `bml:"Driver"`
	Multiplier int     `bml:"Multiplier"`
	Luminance  float64 `bml:"Luminance"`
	ColorBleed bool    `bml:"ColorBleed"`
}

type TestAudioSettings struct {
	Driver  string  `bml:"Driver"`
	Volume  float64 `bml:"Volume"`
	Mute    bool    `bml:"Mute"`
	Latency int64   `bml:"Latency"`
}

type TestSettings struct {
	Video TestVideoSettings `bml:"Video"`
	Audio TestAudioSettings `bml:"Audio"`
}

func TestUnmarshalBasic(t *testing.T) {
	input := `Video
  Driver: Metal
  Multiplier: 2
  Luminance: 1.5
  ColorBleed: true
Audio
  Driver: SDL
  Volume: 0.8
  Mute: false
  Latency: 20`

	var settings TestSettings
	err := Unmarshal([]byte(input), &settings)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if settings.Video.Driver != "Metal" {
		t.Errorf("expected 'Metal', got %q", settings.Video.Driver)
	}
	if settings.Video.Multiplier != 2 {
		t.Errorf("expected 2, got %d", settings.Video.Multiplier)
	}
	if settings.Video.Luminance != 1.5 {
		t.Errorf("expected 1.5, got %f", settings.Video.Luminance)
	}
	if settings.Video.ColorBleed != true {
		t.Error("expected true")
	}
	if settings.Audio.Driver != "SDL" {
		t.Errorf("expected 'SDL', got %q", settings.Audio.Driver)
	}
	if settings.Audio.Volume != 0.8 {
		t.Errorf("expected 0.8, got %f", settings.Audio.Volume)
	}
	if settings.Audio.Mute != false {
		t.Error("expected false")
	}
	if settings.Audio.Latency != 20 {
		t.Errorf("expected 20, got %d", settings.Audio.Latency)
	}
}

func TestUnmarshalMissingNodes(t *testing.T) {
	input := `Video
  Driver: Metal`

	var settings TestSettings
	err := Unmarshal([]byte(input), &settings)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if settings.Video.Driver != "Metal" {
		t.Errorf("expected 'Metal', got %q", settings.Video.Driver)
	}
	// Missing fields should have zero values
	if settings.Video.Multiplier != 0 {
		t.Errorf("expected 0, got %d", settings.Video.Multiplier)
	}
	if settings.Audio.Driver != "" {
		t.Errorf("expected empty, got %q", settings.Audio.Driver)
	}
}

func TestUnmarshalNonPointer(t *testing.T) {
	var settings TestSettings
	err := Unmarshal([]byte("Video"), settings)
	if err == nil {
		t.Fatal("expected error for non-pointer")
	}
	if !strings.Contains(err.Error(), "pointer") {
		t.Errorf("expected 'pointer' in error, got: %v", err)
	}
}

func TestUnmarshalNilPointer(t *testing.T) {
	err := Unmarshal([]byte("Video"), nil)
	if err == nil {
		t.Fatal("expected error for nil pointer")
	}
}

func TestUnmarshalNonStruct(t *testing.T) {
	var s string
	err := Unmarshal([]byte("Video"), &s)
	if err == nil {
		t.Fatal("expected error for non-struct")
	}
	if !strings.Contains(err.Error(), "struct") {
		t.Errorf("expected 'struct' in error, got: %v", err)
	}
}

type TestPointerSettings struct {
	Driver *string `bml:"Driver"`
	Count  *int    `bml:"Count"`
}

func TestUnmarshalPointerFields(t *testing.T) {
	input := `Driver: Metal
Count: 5`

	var settings TestPointerSettings
	err := Unmarshal([]byte(input), &settings)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if settings.Driver == nil || *settings.Driver != "Metal" {
		t.Error("expected Driver to be 'Metal'")
	}
	if settings.Count == nil || *settings.Count != 5 {
		t.Error("expected Count to be 5")
	}
}

func TestUnmarshalPointerFieldsMissing(t *testing.T) {
	input := `Other: value`

	var settings TestPointerSettings
	err := Unmarshal([]byte(input), &settings)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if settings.Driver != nil {
		t.Error("expected Driver to be nil")
	}
	if settings.Count != nil {
		t.Error("expected Count to be nil")
	}
}

type TestUnexportedFields struct {
	Public  string `bml:"Public"`
	private string `bml:"private"`
}

func TestUnmarshalUnexportedFields(t *testing.T) {
	input := `Public: value
private: secret`

	var settings TestUnexportedFields
	err := Unmarshal([]byte(input), &settings)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if settings.Public != "value" {
		t.Errorf("expected 'value', got %q", settings.Public)
	}
	// private field should be zero value (unexported)
	if settings.private != "" {
		t.Errorf("expected empty, got %q", settings.private)
	}
}

type TestNoTagFields struct {
	Tagged   string `bml:"Tagged"`
	Untagged string
}

func TestUnmarshalNoTagFields(t *testing.T) {
	input := `Tagged: value
Untagged: ignored`

	var settings TestNoTagFields
	err := Unmarshal([]byte(input), &settings)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if settings.Tagged != "value" {
		t.Errorf("expected 'value', got %q", settings.Tagged)
	}
	if settings.Untagged != "" {
		t.Errorf("expected empty (no tag), got %q", settings.Untagged)
	}
}

type TestUintFields struct {
	Count  uint   `bml:"Count"`
	Count8 uint8  `bml:"Count8"`
	Count64 uint64 `bml:"Count64"`
}

func TestUnmarshalUintFields(t *testing.T) {
	input := `Count: 42
Count8: 255
Count64: 9999999999`

	var settings TestUintFields
	err := Unmarshal([]byte(input), &settings)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if settings.Count != 42 {
		t.Errorf("expected 42, got %d", settings.Count)
	}
	if settings.Count8 != 255 {
		t.Errorf("expected 255, got %d", settings.Count8)
	}
	if settings.Count64 != 9999999999 {
		t.Errorf("expected 9999999999, got %d", settings.Count64)
	}
}

func TestUnmarshalInvalidInt(t *testing.T) {
	input := `Count: abc`

	type S struct {
		Count int `bml:"Count"`
	}
	var s S
	err := Unmarshal([]byte(input), &s)
	if err == nil {
		t.Fatal("expected error for invalid int")
	}
}

func TestUnmarshalInvalidUint(t *testing.T) {
	input := `Count: -5`

	type S struct {
		Count uint `bml:"Count"`
	}
	var s S
	err := Unmarshal([]byte(input), &s)
	if err == nil {
		t.Fatal("expected error for invalid uint")
	}
}

func TestUnmarshalInvalidFloat(t *testing.T) {
	input := `Value: abc`

	type S struct {
		Value float64 `bml:"Value"`
	}
	var s S
	err := Unmarshal([]byte(input), &s)
	if err == nil {
		t.Fatal("expected error for invalid float")
	}
}

func TestUnmarshalEmptyNumericValues(t *testing.T) {
	input := `Int:
Float:
Uint:`

	type S struct {
		Int   int     `bml:"Int"`
		Float float64 `bml:"Float"`
		Uint  uint    `bml:"Uint"`
	}
	var s S
	err := Unmarshal([]byte(input), &s)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	// Empty values should leave zero values
	if s.Int != 0 || s.Float != 0 || s.Uint != 0 {
		t.Error("expected zero values for empty strings")
	}
}

type TestUnsupportedType struct {
	Data []string `bml:"Data"`
}

func TestUnmarshalUnsupportedType(t *testing.T) {
	input := `Data: value`

	var settings TestUnsupportedType
	err := Unmarshal([]byte(input), &settings)
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("expected 'unsupported' in error, got: %v", err)
	}
}

func TestMarshalBasic(t *testing.T) {
	settings := TestSettings{
		Video: TestVideoSettings{
			Driver:     "Metal",
			Multiplier: 2,
			Luminance:  1.5,
			ColorBleed: true,
		},
		Audio: TestAudioSettings{
			Driver: "SDL",
			Volume: 0.8,
			Mute:   false,
		},
	}

	data, err := Marshal(&settings)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	// Parse it back
	var result TestSettings
	err = Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if result.Video.Driver != "Metal" {
		t.Errorf("expected 'Metal', got %q", result.Video.Driver)
	}
	if result.Video.Multiplier != 2 {
		t.Errorf("expected 2, got %d", result.Video.Multiplier)
	}
}

func TestMarshalNonPointer(t *testing.T) {
	settings := TestSettings{}
	data, err := Marshal(settings) // non-pointer should work
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestMarshalNilPointer(t *testing.T) {
	var settings *TestSettings
	_, err := Marshal(settings)
	if err == nil {
		t.Fatal("expected error for nil pointer")
	}
}

func TestMarshalNonStruct(t *testing.T) {
	s := "string"
	_, err := Marshal(&s)
	if err == nil {
		t.Fatal("expected error for non-struct")
	}
}

func TestMarshalPointerFields(t *testing.T) {
	driver := "Metal"
	count := 5
	settings := TestPointerSettings{
		Driver: &driver,
		Count:  &count,
	}

	data, err := Marshal(&settings)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var result TestPointerSettings
	err = Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if result.Driver == nil || *result.Driver != "Metal" {
		t.Error("expected Driver to be 'Metal'")
	}
}

func TestMarshalNilPointerFields(t *testing.T) {
	settings := TestPointerSettings{
		Driver: nil,
		Count:  nil,
	}

	data, err := Marshal(&settings)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	// Nil pointer fields should be skipped
	if strings.Contains(string(data), "Driver") {
		t.Error("expected nil Driver to be skipped")
	}
}

func TestMarshalUnexportedFields(t *testing.T) {
	settings := TestUnexportedFields{
		Public:  "value",
		private: "secret",
	}

	data, err := Marshal(&settings)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	// private field should not be in output
	if strings.Contains(string(data), "private") {
		t.Error("expected private field to be skipped")
	}
}

func TestMarshalNoTagFields(t *testing.T) {
	settings := TestNoTagFields{
		Tagged:   "value",
		Untagged: "ignored",
	}

	data, err := Marshal(&settings)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	if !strings.Contains(string(data), "Tagged") {
		t.Error("expected Tagged field in output")
	}
	if strings.Contains(string(data), "Untagged") {
		t.Error("expected Untagged field to be skipped (no tag)")
	}
}

func TestMarshalUnsupportedType(t *testing.T) {
	settings := TestUnsupportedType{
		Data: []string{"a", "b"},
	}

	_, err := Marshal(&settings)
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestMarshalUintFields(t *testing.T) {
	settings := TestUintFields{
		Count:   42,
		Count8:  255,
		Count64: 9999999999,
	}

	data, err := Marshal(&settings)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var result TestUintFields
	err = Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if result.Count != 42 || result.Count8 != 255 || result.Count64 != 9999999999 {
		t.Error("uint fields mismatch after round-trip")
	}
}

// === Integration Tests ===

func TestParseRealSettingsFile(t *testing.T) {
	data, err := os.ReadFile("/Users/josediazgonzalez/Library/Application Support/ares/settings.bml")
	if err != nil {
		t.Skipf("skipping: settings.bml not found: %v", err)
	}

	doc, err := Parse(data)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// Verify some known values from the real file
	if driver := doc.Root.Get("Video/Driver").String(""); driver == "" {
		t.Error("expected Video/Driver to have a value")
	}

	if doc.Root.Get("Video/Multiplier").Int(0) == 0 {
		t.Error("expected Video/Multiplier to have a value")
	}

	// Test boolean value
	_ = doc.Root.Get("Boot/Fast").Bool(false)

	// Test float value
	_ = doc.Root.Get("Video/Luminance").Float(0)
}

func TestRoundTripRealSettingsFile(t *testing.T) {
	data, err := os.ReadFile("/Users/josediazgonzalez/Library/Application Support/ares/settings.bml")
	if err != nil {
		t.Skipf("skipping: settings.bml not found: %v", err)
	}

	doc, err := Parse(data)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// Get original values
	origDriver := doc.Root.Get("Video/Driver").String("")
	origMultiplier := doc.Root.Get("Video/Multiplier").Int(0)

	// Serialize and re-parse
	output := Serialize(doc)
	doc2, err := Parse(output)
	if err != nil {
		t.Fatalf("re-parse error: %v", err)
	}

	// Verify values match
	if doc2.Root.Get("Video/Driver").String("") != origDriver {
		t.Error("Video/Driver mismatch after round-trip")
	}
	if doc2.Root.Get("Video/Multiplier").Int(0) != origMultiplier {
		t.Error("Video/Multiplier mismatch after round-trip")
	}
}

func TestModifyAndSerialize(t *testing.T) {
	input := `Video
  Driver: OpenGL
  Multiplier: 1`

	doc, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// Modify values
	doc.Root.Get("Video").Set("Driver", "Metal")
	doc.Root.Get("Video").SetInt("Multiplier", 2)
	doc.Root.Get("Video").SetBool("NewSetting", true)

	// Serialize and re-parse
	output := Serialize(doc)
	doc2, err := Parse(output)
	if err != nil {
		t.Fatalf("re-parse error: %v", err)
	}

	if doc2.Root.Get("Video/Driver").String("") != "Metal" {
		t.Error("expected Driver to be 'Metal'")
	}
	if doc2.Root.Get("Video/Multiplier").Int(0) != 2 {
		t.Error("expected Multiplier to be 2")
	}
	if doc2.Root.Get("Video/NewSetting").Bool(false) != true {
		t.Error("expected NewSetting to be true")
	}
}

// === Helper function tests ===

func TestIsValidNameChar(t *testing.T) {
	valid := []byte{'A', 'Z', 'a', 'z', '0', '9', '-', '.'}
	for _, c := range valid {
		if !isValidNameChar(c) {
			t.Errorf("expected %c to be valid", c)
		}
	}

	invalid := []byte{' ', ':', '=', '"', '\t', '\n', '@', '!'}
	for _, c := range invalid {
		if isValidNameChar(c) {
			t.Errorf("expected %c to be invalid", c)
		}
	}
}

func TestReadDepth(t *testing.T) {
	tests := []struct {
		line     string
		expected int
	}{
		{"Node", 0},
		{"  Node", 2},
		{"\tNode", 1},
		{"\t\tNode", 2},
		{"    Node", 4},
		{"\t  Node", 3},
	}

	for _, tt := range tests {
		depth := readDepth(tt.line)
		if depth != tt.expected {
			t.Errorf("readDepth(%q) = %d, expected %d", tt.line, depth, tt.expected)
		}
	}
}

// === Additional edge case tests for 100% coverage ===

func TestParseValueNoContent(t *testing.T) {
	// Test parseValue with position at end of line
	value, pos, err := parseValue("Node", 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != "" {
		t.Errorf("expected empty value, got %q", value)
	}
	if pos != 4 {
		t.Errorf("expected pos 4, got %d", pos)
	}
}

func TestParseValueUnknownFormat(t *testing.T) {
	// Test parseValue with unknown format (not :, =, or ")
	value, pos, err := parseValue("Node X", 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != "" {
		t.Errorf("expected empty value, got %q", value)
	}
	if pos != 4 {
		t.Errorf("expected pos 4, got %d", pos)
	}
}

func TestSerializeNilNode(t *testing.T) {
	// This shouldn't panic
	serializeNode(nil, 0, nil)
}

func TestNodeGetPathWithEmptyParts(t *testing.T) {
	doc, _ := Parse([]byte("Video\n  Driver: Metal"))

	// Path with empty parts (double slash)
	node := doc.Root.Get("Video//Driver")
	if node == nil {
		t.Fatal("expected to find node with empty path parts")
	}
	if node.Value != "Metal" {
		t.Errorf("expected 'Metal', got %q", node.Value)
	}
}

func TestNodeSetEmptyPath(t *testing.T) {
	doc, _ := Parse([]byte(""))
	result := doc.Root.Set("", "value")
	if result != doc.Root {
		t.Error("expected root node for empty path")
	}
}

func TestDeepEqual(t *testing.T) {
	input := `A
  B
    C: value`

	doc1, _ := Parse([]byte(input))
	doc2, _ := Parse([]byte(input))

	if !reflect.DeepEqual(doc1.Root.Get("A/B/C"), doc2.Root.Get("A/B/C")) {
		t.Error("expected equal nodes")
	}
}

func TestParseColonValueTrailingSpaces(t *testing.T) {
	input := "Driver: Metal   "

	doc, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if doc.Root.Children[0].Value != "Metal" {
		t.Errorf("expected 'Metal', got %q", doc.Root.Children[0].Value)
	}
}

func TestFloat32Field(t *testing.T) {
	input := `Value: 3.14`

	type S struct {
		Value float32 `bml:"Value"`
	}
	var s S
	err := Unmarshal([]byte(input), &s)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if s.Value < 3.13 || s.Value > 3.15 {
		t.Errorf("expected ~3.14, got %f", s.Value)
	}
}

func TestInt8Int16Int32Fields(t *testing.T) {
	input := `I8: 127
I16: 32000
I32: 2000000`

	type S struct {
		I8  int8  `bml:"I8"`
		I16 int16 `bml:"I16"`
		I32 int32 `bml:"I32"`
	}
	var s S
	err := Unmarshal([]byte(input), &s)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if s.I8 != 127 {
		t.Errorf("expected 127, got %d", s.I8)
	}
	if s.I16 != 32000 {
		t.Errorf("expected 32000, got %d", s.I16)
	}
	if s.I32 != 2000000 {
		t.Errorf("expected 2000000, got %d", s.I32)
	}
}

func TestUint16Uint32Fields(t *testing.T) {
	input := `U16: 65000
U32: 4000000`

	type S struct {
		U16 uint16 `bml:"U16"`
		U32 uint32 `bml:"U32"`
	}
	var s S
	err := Unmarshal([]byte(input), &s)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if s.U16 != 65000 {
		t.Errorf("expected 65000, got %d", s.U16)
	}
	if s.U32 != 4000000 {
		t.Errorf("expected 4000000, got %d", s.U32)
	}
}

func TestMarshalFloat32(t *testing.T) {
	type S struct {
		Value float32 `bml:"Value"`
	}
	s := S{Value: 3.14}
	data, err := Marshal(&s)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if !strings.Contains(string(data), "3.14") {
		t.Errorf("expected '3.14' in output, got %q", string(data))
	}
}

func TestMarshalIntVariants(t *testing.T) {
	type S struct {
		I8  int8  `bml:"I8"`
		I16 int16 `bml:"I16"`
		I32 int32 `bml:"I32"`
	}
	s := S{I8: 10, I16: 1000, I32: 100000}
	data, err := Marshal(&s)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var result S
	err = Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if result.I8 != 10 || result.I16 != 1000 || result.I32 != 100000 {
		t.Error("int variant mismatch after round-trip")
	}
}

func TestMarshalUintVariants(t *testing.T) {
	type S struct {
		U8  uint8  `bml:"U8"`
		U16 uint16 `bml:"U16"`
		U32 uint32 `bml:"U32"`
	}
	s := S{U8: 200, U16: 60000, U32: 4000000}
	data, err := Marshal(&s)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var result S
	err = Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if result.U8 != 200 || result.U16 != 60000 || result.U32 != 4000000 {
		t.Error("uint variant mismatch after round-trip")
	}
}

// === Additional edge case tests for 100% coverage ===

func TestUnmarshalParseError(t *testing.T) {
	// Invalid BML that causes Parse to fail
	input := `Driver="unclosed`

	type S struct {
		Driver string `bml:"Driver"`
	}
	var s S
	err := Unmarshal([]byte(input), &s)
	if err == nil {
		t.Fatal("expected error for invalid BML")
	}
}

func TestRemovePathWithEmptyParts(t *testing.T) {
	doc, _ := Parse([]byte("Video\n  Driver: Metal"))

	// Path with empty parts
	removed := doc.Root.Remove("Video//Driver")
	if !removed {
		t.Error("expected Remove to handle empty path parts")
	}
}

func TestUnmarshalNodeNil(t *testing.T) {
	// Test unmarshalNode with nil node directly
	type S struct {
		Value string `bml:"Value"`
	}
	input := ""
	var s S
	err := Unmarshal([]byte(input), &s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSerializeNodeWithChildrenAndMultilineValue(t *testing.T) {
	// Node with both multiline value AND children
	doc := &Document{Root: &Node{
		Children: []*Node{
			{
				Name:  "Desc",
				Value: "Line1\nLine2",
				Children: []*Node{
					{Name: "Child", Value: "value"},
				},
			},
		},
	}}
	data := Serialize(doc)
	// Should serialize without panic
	if len(data) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestUnmarshalNodeError(t *testing.T) {
	// Test error propagation in unmarshalNode
	input := `Nested
  Value: abc`

	type Inner struct {
		Value int `bml:"Value"`
	}
	type S struct {
		Nested Inner `bml:"Nested"`
	}
	var s S
	err := Unmarshal([]byte(input), &s)
	if err == nil {
		t.Fatal("expected error for invalid nested int")
	}
}

func TestMarshalStructError(t *testing.T) {
	// Test error in nested struct marshaling
	type Inner struct {
		Data []string `bml:"Data"`
	}
	type S struct {
		Nested Inner `bml:"Nested"`
	}
	s := S{Nested: Inner{Data: []string{"a"}}}
	_, err := Marshal(&s)
	if err == nil {
		t.Fatal("expected error for unsupported type in nested struct")
	}
}

func TestParseNodeAttributeNoValue(t *testing.T) {
	// Attribute without a value
	input := "Node attr1"
	doc, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(doc.Root.Children[0].Children) != 1 {
		t.Fatalf("expected 1 attribute, got %d", len(doc.Root.Children[0].Children))
	}
	if doc.Root.Children[0].Children[0].Name != "attr1" {
		t.Errorf("expected attr1, got %q", doc.Root.Children[0].Children[0].Name)
	}
}

func TestMultilineValueWithExistingValue(t *testing.T) {
	// Node with initial value followed by continuation
	input := `Desc: Initial
  : Line2`

	doc, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	node := doc.Root.Children[0]
	expected := "Initial\nLine2"
	if node.Value != expected {
		t.Errorf("expected %q, got %q", expected, node.Value)
	}
}

func TestUnmarshalNilPointerTyped(t *testing.T) {
	type S struct {
		Value string `bml:"Value"`
	}
	var s *S = nil
	err := Unmarshal([]byte("Value: test"), s)
	if err == nil {
		t.Fatal("expected error for nil typed pointer")
	}
}

func TestUnmarshalNodeFieldError(t *testing.T) {
	// Test when unmarshalValue returns an error for a field
	input := `Value: notanumber`

	type S struct {
		Value int `bml:"Value"`
	}
	var s S
	err := Unmarshal([]byte(input), &s)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Value") {
		t.Errorf("expected error to mention field name, got: %v", err)
	}
}

// Test internal parseNode functions by calling them via exported wrappers
// These test defensive code paths

func TestParseNodeEdgeCases(t *testing.T) {
	// Test calling parseNode directly to trigger defensive checks

	// Test "unexpected end of input"
	lines := []string{}
	index := 0
	_, err := parseNode(lines, &index, -1)
	if err == nil {
		t.Fatal("expected error for empty lines")
	}
	if !strings.Contains(err.Error(), "unexpected end") {
		t.Errorf("expected 'unexpected end' error, got: %v", err)
	}

	// Test "invalid indentation" - node at same or lower depth than parent
	lines = []string{"Node", "  Child"}
	index = 1 // Start at Child
	_, err = parseNode(lines, &index, 5) // Parent depth 5, but Child has depth 2
	if err == nil {
		t.Fatal("expected error for invalid indentation")
	}
	if !strings.Contains(err.Error(), "invalid indentation") {
		t.Errorf("expected 'invalid indentation' error, got: %v", err)
	}
}

func TestNormalizeLinesThoroughly(t *testing.T) {
	// Test various line ending combinations
	tests := []struct {
		input    string
		expected int // expected number of lines after normalization
	}{
		{"A\r\nB\r\nC", 3},        // Windows
		{"A\rB\rC", 3},            // Old Mac
		{"A\nB\nC", 3},            // Unix
		{"A\n\nB", 2},             // Empty lines removed
		{"// comment\nA", 1},      // Comment removed
		{"  // comment\nA", 1},    // Indented comment removed
		{"\t// comment\nA", 1},    // Tab-indented comment removed
		{"", 0},                   // Empty
		{"   \n\t\n  ", 0},        // Only whitespace
	}

	for _, tt := range tests {
		lines := normalizeLines(tt.input)
		if len(lines) != tt.expected {
			t.Errorf("normalizeLines(%q) = %d lines, expected %d", tt.input, len(lines), tt.expected)
		}
	}
}

func TestParseNodeAttributeEdgeCases(t *testing.T) {
	// Test attribute parsing edge cases

	// Node with trailing spaces after value (pos >= len after spaces)
	input := "Node: value   "
	doc, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.Root.Children[0].Value != "value" {
		t.Errorf("expected 'value', got %q", doc.Root.Children[0].Value)
	}

	// Node with invalid character after attributes (breaks at attrStart)
	input = "Node attr1=v1 !"
	doc, err = Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should parse attr1 but stop at !
	if len(doc.Root.Children[0].Children) != 1 {
		t.Errorf("expected 1 attribute, got %d", len(doc.Root.Children[0].Children))
	}
}

func TestParseNodeChildError(t *testing.T) {
	// Test error propagation when parsing child node fails
	// We need a child node that causes an error
	// The only way to cause an error in child parsing is unclosed quote

	input := `Parent
  Child="unclosed`

	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error from child parsing")
	}
}

func TestUnmarshalNodeNilDirectly(t *testing.T) {
	// Test unmarshalNode when called with nil node
	type S struct {
		Value string `bml:"Value"`
	}
	var s S
	// Call unmarshalNode directly with nil
	err := unmarshalNode(nil, reflect.ValueOf(&s).Elem())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseNodeAttributeTrailingSpaces(t *testing.T) {
	// Test attribute loop ending at pos >= len(line) after spaces
	input := "Node attr1=v1 "
	doc, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(doc.Root.Children[0].Children) != 1 {
		t.Errorf("expected 1 attribute, got %d", len(doc.Root.Children[0].Children))
	}
}

func TestParseNodeAttributeParseValueError(t *testing.T) {
	// Test error in parseValue during attribute parsing
	input := `Node attr="unclosed`
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("expected error for unclosed attribute quote")
	}
}

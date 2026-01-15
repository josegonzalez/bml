// Package bml provides parsing and serialization for BML (Binary Markup Language) files.
// BML is a hierarchical markup format used by the ares emulator for configuration files.
package bml

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Node represents a BML node with a name, value, and children.
type Node struct {
	Name     string
	Value    string
	Children []*Node
}

// Document represents a parsed BML document.
type Document struct {
	Root *Node // Anonymous root containing top-level nodes
}

// Parse parses BML data and returns a Document.
func Parse(data []byte) (*Document, error) {
	lines := normalizeLines(string(data))
	if len(lines) == 0 {
		return &Document{Root: &Node{}}, nil
	}

	root := &Node{}
	index := 0

	for index < len(lines) {
		node, err := parseNode(lines, &index, -1)
		if err != nil {
			return nil, err
		}
		root.Children = append(root.Children, node)
	}

	return &Document{Root: root}, nil
}

// normalizeLines converts the input into a slice of non-empty, non-comment lines.
func normalizeLines(input string) []string {
	// Normalize line endings
	input = strings.ReplaceAll(input, "\r\n", "\n")
	input = strings.ReplaceAll(input, "\r", "\n")

	rawLines := strings.Split(input, "\n")
	var lines []string

	for _, line := range rawLines {
		// Skip empty lines (but preserve lines that are only whitespace for indentation tracking)
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Skip comment lines
		depth := readDepth(line)
		rest := line[depth:]
		if strings.HasPrefix(rest, "//") {
			continue
		}

		lines = append(lines, line)
	}

	return lines
}

// readDepth counts the leading whitespace characters (tabs or spaces).
func readDepth(line string) int {
	depth := 0
	for _, c := range line {
		if c == '\t' || c == ' ' {
			depth++
		} else {
			break
		}
	}
	return depth
}

// isValidNameChar returns true if c is a valid BML name character (A-Z, a-z, 0-9, -, .)
func isValidNameChar(c byte) bool {
	return (c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '.'
}

// parseNode parses a single node and its children from the lines.
func parseNode(lines []string, index *int, parentDepth int) (*Node, error) {
	if *index >= len(lines) {
		return nil, errors.New("unexpected end of input")
	}

	line := lines[*index]
	*index++

	depth := readDepth(line)
	if depth <= parentDepth && parentDepth >= 0 {
		return nil, fmt.Errorf("invalid indentation at line: %s", line)
	}

	pos := depth
	node := &Node{}

	// Parse name
	nameStart := pos
	for pos < len(line) && isValidNameChar(line[pos]) {
		pos++
	}
	if pos == nameStart {
		return nil, fmt.Errorf("invalid node name at line: %s", line)
	}
	node.Name = line[nameStart:pos]

	// Parse value
	if pos < len(line) {
		value, newPos, err := parseValue(line, pos)
		if err != nil {
			return nil, err
		}
		node.Value = value
		pos = newPos
	}

	// Parse attributes (space-separated key-value pairs on the same line)
	for pos < len(line) {
		// Skip spaces
		for pos < len(line) && line[pos] == ' ' {
			pos++
		}
		if pos >= len(line) {
			break
		}

		// Check for inline comment
		if pos+1 < len(line) && line[pos:pos+2] == "//" {
			break
		}

		// Parse attribute name
		attrStart := pos
		for pos < len(line) && isValidNameChar(line[pos]) {
			pos++
		}
		if pos == attrStart {
			break
		}
		attrName := line[attrStart:pos]

		// Parse attribute value
		attrValue := ""
		if pos < len(line) {
			var err error
			attrValue, pos, err = parseValue(line, pos)
			if err != nil {
				return nil, err
			}
		}

		node.Children = append(node.Children, &Node{Name: attrName, Value: attrValue})
	}

	// Parse child nodes based on indentation
	for *index < len(lines) {
		childDepth := readDepth(lines[*index])
		if childDepth <= depth {
			break
		}

		// Check for multiline value continuation (line starting with : at deeper depth)
		rest := strings.TrimLeft(lines[*index], " \t")
		if strings.HasPrefix(rest, ":") {
			// Multiline value continuation
			continuation := strings.TrimPrefix(rest, ":")
			continuation = strings.TrimPrefix(continuation, " ") // Trim one leading space if present
			if node.Value != "" {
				node.Value += "\n"
			}
			node.Value += continuation
			*index++
			continue
		}

		child, err := parseNode(lines, index, depth)
		if err != nil {
			return nil, err
		}
		node.Children = append(node.Children, child)
	}

	return node, nil
}

// parseValue parses a value starting at pos in line. Returns the value, new position, and any error.
func parseValue(line string, pos int) (string, int, error) {
	if pos >= len(line) {
		return "", pos, nil
	}

	switch line[pos] {
	case ':':
		// Colon format: Name: value
		pos++
		// Skip one leading space if present
		if pos < len(line) && line[pos] == ' ' {
			pos++
		}
		// Value extends to end of line (or until inline comment)
		end := pos
		for end < len(line) {
			if end+1 < len(line) && line[end:end+2] == "//" {
				break
			}
			end++
		}
		value := strings.TrimRight(line[pos:end], " ")
		return value, end, nil

	case '=':
		pos++
		if pos >= len(line) {
			return "", pos, nil
		}

		if line[pos] == '"' {
			// Quoted format: Name="value"
			pos++
			end := pos
			for end < len(line) && line[end] != '"' {
				end++
			}
			if end >= len(line) {
				return "", pos, fmt.Errorf("unclosed quote in line: %s", line)
			}
			value := line[pos:end]
			return value, end + 1, nil
		}

		// Unquoted format: Name=value (no spaces allowed)
		end := pos
		for end < len(line) && line[end] != ' ' && line[end] != '"' {
			end++
		}
		value := line[pos:end]
		return value, end, nil

	default:
		return "", pos, nil
	}
}

// Get retrieves a child node by path (e.g., "Video/Driver").
// Returns nil if the path doesn't exist.
func (n *Node) Get(path string) *Node {
	if n == nil {
		return nil
	}

	parts := strings.Split(path, "/")
	current := n

	for _, part := range parts {
		if part == "" {
			continue
		}

		found := false
		for _, child := range current.Children {
			if child.Name == part {
				current = child
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}

	return current
}

// String returns the node's value as a string, or the fallback if the node is nil.
func (n *Node) String(fallback string) string {
	if n == nil {
		return fallback
	}
	return strings.TrimSpace(n.Value)
}

// Bool returns the node's value as a boolean, or the fallback if the node is nil or not a valid bool.
func (n *Node) Bool(fallback bool) bool {
	if n == nil {
		return fallback
	}
	v := strings.TrimSpace(n.Value)
	if v == "true" {
		return true
	}
	if v == "false" {
		return false
	}
	return fallback
}

// Int returns the node's value as an integer, or the fallback if the node is nil or not a valid int.
func (n *Node) Int(fallback int) int {
	if n == nil {
		return fallback
	}
	v := strings.TrimSpace(n.Value)
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}

// Float returns the node's value as a float64, or the fallback if the node is nil or not a valid float.
func (n *Node) Float(fallback float64) float64 {
	if n == nil {
		return fallback
	}
	v := strings.TrimSpace(n.Value)
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}

// Set sets or creates a node at the given path with the given value.
// Creates intermediate nodes as needed. Returns the node that was set.
func (n *Node) Set(path string, value string) *Node {
	if n == nil {
		return nil
	}

	parts := strings.Split(path, "/")
	current := n

	for i, part := range parts {
		if part == "" {
			continue
		}

		var found *Node
		for _, child := range current.Children {
			if child.Name == part {
				found = child
				break
			}
		}

		if found == nil {
			found = &Node{Name: part}
			current.Children = append(current.Children, found)
		}

		if i == len(parts)-1 {
			found.Value = value
			return found
		}

		current = found
	}

	return current
}

// SetBool sets a boolean value at the given path.
func (n *Node) SetBool(path string, value bool) *Node {
	if value {
		return n.Set(path, "true")
	}
	return n.Set(path, "false")
}

// SetInt sets an integer value at the given path.
func (n *Node) SetInt(path string, value int) *Node {
	return n.Set(path, strconv.Itoa(value))
}

// SetFloat sets a float value at the given path.
func (n *Node) SetFloat(path string, value float64) *Node {
	return n.Set(path, strconv.FormatFloat(value, 'f', -1, 64))
}

// Remove removes a child node at the given path. Returns true if the node was removed.
func (n *Node) Remove(path string) bool {
	if n == nil {
		return false
	}

	parts := strings.Split(path, "/")

	// Navigate to the parent of the node to remove
	current := n
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		if part == "" {
			continue
		}

		found := false
		for _, child := range current.Children {
			if child.Name == part {
				current = child
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Remove the last node in the path
	targetName := parts[len(parts)-1]
	for i, child := range current.Children {
		if child.Name == targetName {
			current.Children = append(current.Children[:i], current.Children[i+1:]...)
			return true
		}
	}

	return false
}

// Serialize converts a Document back to BML format.
func Serialize(doc *Document) []byte {
	if doc == nil || doc.Root == nil {
		return nil
	}

	var buf bytes.Buffer
	for _, child := range doc.Root.Children {
		serializeNode(child, 0, &buf)
	}
	return buf.Bytes()
}

// serializeNode writes a node and its children to the buffer.
func serializeNode(node *Node, depth int, buf *bytes.Buffer) {
	if node == nil {
		return
	}

	// Write indentation
	for i := 0; i < depth*2; i++ {
		buf.WriteByte(' ')
	}

	// Write name
	buf.WriteString(node.Name)

	// Write value
	if node.Value != "" {
		// Check for multiline values
		if strings.Contains(node.Value, "\n") {
			buf.WriteByte('\n')
			lines := strings.Split(node.Value, "\n")
			for _, line := range lines {
				for i := 0; i < (depth+1)*2; i++ {
					buf.WriteByte(' ')
				}
				buf.WriteString(": ")
				buf.WriteString(line)
				buf.WriteByte('\n')
			}
		} else {
			buf.WriteString(": ")
			buf.WriteString(node.Value)
			buf.WriteByte('\n')
		}
	} else {
		buf.WriteByte('\n')
	}

	// Write children (skip if we just wrote multiline value)
	if !strings.Contains(node.Value, "\n") || node.Value == "" {
		for _, child := range node.Children {
			serializeNode(child, depth+1, buf)
		}
	} else {
		// For multiline values, children come after the value lines
		for _, child := range node.Children {
			serializeNode(child, depth+1, buf)
		}
	}
}

// Unmarshal parses BML data and populates the struct pointed to by v.
func Unmarshal(data []byte, v interface{}) error {
	doc, err := Parse(data)
	if err != nil {
		return err
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return errors.New("bml: Unmarshal requires a pointer")
	}
	if rv.IsNil() {
		return errors.New("bml: Unmarshal requires a non-nil pointer")
	}

	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return errors.New("bml: Unmarshal requires a pointer to a struct")
	}

	return unmarshalNode(doc.Root, rv)
}

// unmarshalNode populates a struct value from a BML node.
func unmarshalNode(node *Node, v reflect.Value) error {
	if node == nil {
		return nil
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Get the bml tag
		tag := fieldType.Tag.Get("bml")
		if tag == "" {
			continue
		}

		// Find the corresponding BML node
		childNode := node.Get(tag)

		if err := unmarshalValue(childNode, field); err != nil {
			return fmt.Errorf("field %s: %w", fieldType.Name, err)
		}
	}

	return nil
}

// unmarshalValue sets a reflect.Value from a BML node.
func unmarshalValue(node *Node, v reflect.Value) error {
	// Handle pointer types
	if v.Kind() == reflect.Ptr {
		if node == nil {
			return nil // Leave as nil
		}
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		return unmarshalValue(node, v.Elem())
	}

	if node == nil {
		return nil // Leave as zero value
	}

	switch v.Kind() {
	case reflect.String:
		v.SetString(strings.TrimSpace(node.Value))

	case reflect.Bool:
		val := strings.TrimSpace(node.Value)
		v.SetBool(val == "true")

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val := strings.TrimSpace(node.Value)
		if val == "" {
			return nil
		}
		i, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse %q as int: %w", val, err)
		}
		v.SetInt(i)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val := strings.TrimSpace(node.Value)
		if val == "" {
			return nil
		}
		u, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse %q as uint: %w", val, err)
		}
		v.SetUint(u)

	case reflect.Float32, reflect.Float64:
		val := strings.TrimSpace(node.Value)
		if val == "" {
			return nil
		}
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return fmt.Errorf("cannot parse %q as float: %w", val, err)
		}
		v.SetFloat(f)

	case reflect.Struct:
		return unmarshalNode(node, v)

	default:
		return fmt.Errorf("unsupported type: %s", v.Kind())
	}

	return nil
}

// Marshal converts a struct to BML format.
func Marshal(v interface{}) ([]byte, error) {
	rv := reflect.ValueOf(v)

	// Dereference pointer if needed
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, errors.New("bml: Marshal requires a non-nil value")
		}
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return nil, errors.New("bml: Marshal requires a struct or pointer to struct")
	}

	root := &Node{}
	if err := marshalStruct(rv, root); err != nil {
		return nil, err
	}

	return Serialize(&Document{Root: root}), nil
}

// marshalStruct converts a struct to BML nodes and adds them as children of parent.
func marshalStruct(v reflect.Value, parent *Node) error {
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Skip unexported fields
		if !fieldType.IsExported() {
			continue
		}

		// Get the bml tag
		tag := fieldType.Tag.Get("bml")
		if tag == "" {
			continue
		}

		node, err := marshalValue(field, tag)
		if err != nil {
			return fmt.Errorf("field %s: %w", fieldType.Name, err)
		}
		if node != nil {
			parent.Children = append(parent.Children, node)
		}
	}

	return nil
}

// marshalValue converts a reflect.Value to a BML node.
func marshalValue(v reflect.Value, name string) (*Node, error) {
	// Handle pointer types
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, nil // Skip nil pointers
		}
		return marshalValue(v.Elem(), name)
	}

	node := &Node{Name: name}

	switch v.Kind() {
	case reflect.String:
		node.Value = v.String()

	case reflect.Bool:
		if v.Bool() {
			node.Value = "true"
		} else {
			node.Value = "false"
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		node.Value = strconv.FormatInt(v.Int(), 10)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		node.Value = strconv.FormatUint(v.Uint(), 10)

	case reflect.Float32, reflect.Float64:
		node.Value = strconv.FormatFloat(v.Float(), 'f', -1, 64)

	case reflect.Struct:
		if err := marshalStruct(v, node); err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("unsupported type: %s", v.Kind())
	}

	return node, nil
}

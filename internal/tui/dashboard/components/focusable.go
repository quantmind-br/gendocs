package components

import tea "github.com/charmbracelet/bubbletea"

// Focusable represents a UI component that can receive and lose focus.
// This interface allows managing focus state across heterogeneous input types
// (textinput, dropdown, masked input, button, etc.) uniformly via a slice.
//
// All form components in this package implement Focusable, allowing sections
// to manage focus navigation with a simple slice instead of hardcoded switch
// statements on field indices.
type Focusable interface {
	// Focus activates this component. Returns a command if needed (e.g., cursor blink).
	Focus() tea.Cmd

	// Blur deactivates this component.
	Blur()

	// Focused returns true if this component currently has focus.
	Focused() bool

	// View renders the component.
	View() string

	// IsDirty returns true if the value has changed from its original.
	IsDirty() bool
}

// FocusableInput extends Focusable with value get/set for input components.
// Use this for components that hold user-editable values (text fields, dropdowns).
// Buttons and other non-input components only need to implement Focusable.
type FocusableInput interface {
	Focusable

	// Value returns the current value as string.
	Value() string

	// SetValue sets the value from string (for initialization from config).
	SetValue(string)
}

// FocusableSlice manages a slice of focusable components with focus navigation.
// This replaces the brittle switch-on-focusIndex pattern with data-driven focus management.
type FocusableSlice struct {
	items      []FocusableUpdater
	focusIndex int
}

// FocusableUpdater wraps a component to provide a unified Update interface.
// This is necessary because Go interfaces can't express "returns same type as receiver".
type FocusableUpdater interface {
	Focusable
	// Update handles a message and returns a command. The updater modifies itself in place.
	Update(msg tea.Msg) tea.Cmd
}

// NewFocusableSlice creates a new slice from the given components.
func NewFocusableSlice(items ...FocusableUpdater) *FocusableSlice {
	return &FocusableSlice{
		items:      items,
		focusIndex: 0,
	}
}

// Len returns the number of items.
func (fs *FocusableSlice) Len() int {
	return len(fs.items)
}

// Current returns the currently focused item.
func (fs *FocusableSlice) Current() FocusableUpdater {
	if fs.focusIndex >= 0 && fs.focusIndex < len(fs.items) {
		return fs.items[fs.focusIndex]
	}
	return nil
}

// Index returns the current focus index.
func (fs *FocusableSlice) Index() int {
	return fs.focusIndex
}

// FocusNext moves focus to the next component, wrapping around.
func (fs *FocusableSlice) FocusNext() tea.Cmd {
	if len(fs.items) == 0 {
		return nil
	}
	fs.items[fs.focusIndex].Blur()
	fs.focusIndex = (fs.focusIndex + 1) % len(fs.items)
	return fs.items[fs.focusIndex].Focus()
}

// FocusPrev moves focus to the previous component, wrapping around.
func (fs *FocusableSlice) FocusPrev() tea.Cmd {
	if len(fs.items) == 0 {
		return nil
	}
	fs.items[fs.focusIndex].Blur()
	fs.focusIndex--
	if fs.focusIndex < 0 {
		fs.focusIndex = len(fs.items) - 1
	}
	return fs.items[fs.focusIndex].Focus()
}

// FocusFirst focuses the first component.
func (fs *FocusableSlice) FocusFirst() tea.Cmd {
	if len(fs.items) == 0 {
		return nil
	}
	fs.BlurAll()
	fs.focusIndex = 0
	return fs.items[0].Focus()
}

// FocusLast focuses the last component.
func (fs *FocusableSlice) FocusLast() tea.Cmd {
	if len(fs.items) == 0 {
		return nil
	}
	fs.BlurAll()
	fs.focusIndex = len(fs.items) - 1
	return fs.items[fs.focusIndex].Focus()
}

// BlurAll removes focus from all components.
func (fs *FocusableSlice) BlurAll() {
	for _, item := range fs.items {
		item.Blur()
	}
}

// UpdateCurrent sends a message to the currently focused component.
func (fs *FocusableSlice) UpdateCurrent(msg tea.Msg) tea.Cmd {
	if fs.focusIndex >= 0 && fs.focusIndex < len(fs.items) {
		return fs.items[fs.focusIndex].Update(msg)
	}
	return nil
}

// IsDirty returns true if any component has been modified.
func (fs *FocusableSlice) IsDirty() bool {
	for _, item := range fs.items {
		if item.IsDirty() {
			return true
		}
	}
	return false
}

// Get returns the item at the given index.
func (fs *FocusableSlice) Get(index int) FocusableUpdater {
	if index >= 0 && index < len(fs.items) {
		return fs.items[index]
	}
	return nil
}

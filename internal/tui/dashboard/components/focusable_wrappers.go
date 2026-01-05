package components

import tea "github.com/charmbracelet/bubbletea"

// TextFieldWrapper wraps TextFieldModel to implement FocusableUpdater.
type TextFieldWrapper struct {
	*TextFieldModel
}

func WrapTextField(m *TextFieldModel) *TextFieldWrapper {
	return &TextFieldWrapper{TextFieldModel: m}
}

func (w *TextFieldWrapper) Update(msg tea.Msg) tea.Cmd {
	updated, cmd := w.TextFieldModel.Update(msg)
	*w.TextFieldModel = updated
	return cmd
}

// DropdownWrapper wraps DropdownModel to implement FocusableUpdater.
type DropdownWrapper struct {
	*DropdownModel
}

func WrapDropdown(m *DropdownModel) *DropdownWrapper {
	return &DropdownWrapper{DropdownModel: m}
}

func (w *DropdownWrapper) Update(msg tea.Msg) tea.Cmd {
	updated, cmd := w.DropdownModel.Update(msg)
	*w.DropdownModel = updated
	return cmd
}

// MaskedInputWrapper wraps MaskedInputModel to implement FocusableUpdater.
type MaskedInputWrapper struct {
	*MaskedInputModel
}

func WrapMaskedInput(m *MaskedInputModel) *MaskedInputWrapper {
	return &MaskedInputWrapper{MaskedInputModel: m}
}

func (w *MaskedInputWrapper) Update(msg tea.Msg) tea.Cmd {
	updated, cmd := w.MaskedInputModel.Update(msg)
	*w.MaskedInputModel = updated
	return cmd
}

// ButtonWrapper wraps ButtonModel to implement FocusableUpdater.
type ButtonWrapper struct {
	*ButtonModel
}

func WrapButton(m *ButtonModel) *ButtonWrapper {
	return &ButtonWrapper{ButtonModel: m}
}

func (w *ButtonWrapper) Update(msg tea.Msg) tea.Cmd {
	updated, cmd := w.ButtonModel.Update(msg)
	*w.ButtonModel = updated
	return cmd
}

// ToggleWrapper wraps ToggleModel to implement FocusableUpdater.
type ToggleWrapper struct {
	*ToggleModel
}

func WrapToggle(m *ToggleModel) *ToggleWrapper {
	return &ToggleWrapper{ToggleModel: m}
}

func (w *ToggleWrapper) Update(msg tea.Msg) tea.Cmd {
	updated, cmd := w.ToggleModel.Update(msg)
	*w.ToggleModel = updated
	return cmd
}

// Package menu provides an interactive menu system for the EZIO-G500 display.
package menu

import (
	"github.com/sagostin/ezio-g500/pkg/display"
	"github.com/sagostin/ezio-g500/pkg/eziog500"
	"github.com/sagostin/ezio-g500/pkg/font"
)

// MenuItem represents a single menu item.
type MenuItem struct {
	Label    string
	Action   func() error  // Action to execute when selected
	SubMenu  *Menu         // Optional submenu
	Value    func() string // Optional dynamic value display
	Disabled bool          // If true, item cannot be selected
}

// Menu represents an interactive menu.
type Menu struct {
	Title        string
	Items        []MenuItem
	Parent       *Menu
	selected     int
	scrollOffset int
	maxVisible   int
}

// NewMenu creates a new menu with the given title and items.
func NewMenu(title string, items []MenuItem) *Menu {
	return &Menu{
		Title:      title,
		Items:      items,
		maxVisible: 6, // Default for 8-line display minus title and status
	}
}

// SetMaxVisible sets the maximum number of visible items.
func (m *Menu) SetMaxVisible(n int) {
	m.maxVisible = n
}

// AddItem adds an item to the menu.
func (m *Menu) AddItem(item MenuItem) {
	m.Items = append(m.Items, item)
}

// AddSubMenu adds a submenu item.
func (m *Menu) AddSubMenu(label string, subMenu *Menu) {
	subMenu.Parent = m
	m.Items = append(m.Items, MenuItem{
		Label:   label + " >",
		SubMenu: subMenu,
	})
}

// Selected returns the currently selected index.
func (m *Menu) Selected() int {
	return m.selected
}

// SelectNext moves selection to the next item.
func (m *Menu) SelectNext() {
	for {
		m.selected++
		if m.selected >= len(m.Items) {
			m.selected = 0
		}
		// Skip disabled items
		if !m.Items[m.selected].Disabled {
			break
		}
		// Prevent infinite loop if all items are disabled
		if m.allDisabled() {
			break
		}
	}
	m.updateScroll()
}

// SelectPrevious moves selection to the previous item.
func (m *Menu) SelectPrevious() {
	for {
		m.selected--
		if m.selected < 0 {
			m.selected = len(m.Items) - 1
		}
		if !m.Items[m.selected].Disabled {
			break
		}
		if m.allDisabled() {
			break
		}
	}
	m.updateScroll()
}

func (m *Menu) allDisabled() bool {
	for _, item := range m.Items {
		if !item.Disabled {
			return false
		}
	}
	return true
}

func (m *Menu) updateScroll() {
	// Scroll down if selected is below visible area
	if m.selected >= m.scrollOffset+m.maxVisible {
		m.scrollOffset = m.selected - m.maxVisible + 1
	}
	// Scroll up if selected is above visible area
	if m.selected < m.scrollOffset {
		m.scrollOffset = m.selected
	}
}

// Execute runs the action of the currently selected item.
// Returns the submenu if one exists, nil otherwise.
func (m *Menu) Execute() (*Menu, error) {
	if m.selected >= 0 && m.selected < len(m.Items) {
		item := m.Items[m.selected]
		if item.Disabled {
			return nil, nil
		}
		if item.SubMenu != nil {
			return item.SubMenu, nil
		}
		if item.Action != nil {
			return nil, item.Action()
		}
	}
	return nil, nil
}

// Render draws the menu to the display.
func (m *Menu) Render(d *display.Display) error {
	fb := d.FrameBuffer()
	fb.Clear()

	f := font.BuiltinFont
	lineHeight := f.Height()

	// Draw title bar (inverted)
	font.RenderTextInverted(fb, f, 0, 0, m.Title)

	// Draw menu items
	y := lineHeight
	endIdx := m.scrollOffset + m.maxVisible
	if endIdx > len(m.Items) {
		endIdx = len(m.Items)
	}

	for i := m.scrollOffset; i < endIdx; i++ {
		item := m.Items[i]

		text := item.Label
		if item.Value != nil {
			val := item.Value()
			if val != "" {
				text = item.Label + ": " + val
			}
		}

		if i == m.selected {
			// Draw selected item inverted
			fb.FillRect(0, y, eziog500.Width, lineHeight, true)
			// Render text in "off" pixels
			curX := 2
			for _, r := range text {
				glyph := f.GetGlyph(r)
				if glyph == nil {
					continue
				}
				for col, b := range glyph {
					for bit := 0; bit < 8; bit++ {
						if (b & (1 << bit)) != 0 {
							fb.SetPixel(curX+col, y+bit, false)
						}
					}
				}
				curX += len(glyph)
			}
		} else {
			// Normal item
			prefix := "  "
			if item.Disabled {
				prefix = "- "
			}
			font.RenderText(fb, f, 0, y, prefix+text)
		}
		y += lineHeight
	}

	// Draw scroll indicators if needed
	if m.scrollOffset > 0 {
		// Up arrow indicator
		fb.SetPixel(124, lineHeight+2, true)
		fb.SetPixel(125, lineHeight+1, true)
		fb.SetPixel(126, lineHeight+2, true)
	}
	if endIdx < len(m.Items) {
		// Down arrow indicator
		lastY := lineHeight + m.maxVisible*lineHeight - 3
		fb.SetPixel(124, lastY, true)
		fb.SetPixel(125, lastY+1, true)
		fb.SetPixel(126, lastY, true)
	}

	return d.Update()
}

// MenuController manages menu navigation with button input.
type MenuController struct {
	display      *display.Display
	buttonReader *eziog500.ButtonReader
	currentMenu  *Menu
	rootMenu     *Menu
}

// NewMenuController creates a menu controller.
func NewMenuController(d *display.Display, br *eziog500.ButtonReader, rootMenu *Menu) *MenuController {
	return &MenuController{
		display:      d,
		buttonReader: br,
		currentMenu:  rootMenu,
		rootMenu:     rootMenu,
	}
}

// Run starts the menu controller loop.
// It blocks until the menu is exited (by returning from root menu).
func (mc *MenuController) Run() error {
	buttons, stop := mc.buttonReader.ButtonChannel()
	defer stop()

	// Initial render
	if err := mc.currentMenu.Render(mc.display); err != nil {
		return err
	}

	for btn := range buttons {
		needsRender := true

		switch btn {
		case eziog500.ButtonUp:
			mc.currentMenu.SelectPrevious()

		case eziog500.ButtonDown:
			mc.currentMenu.SelectNext()

		case eziog500.ButtonEnter, eziog500.ButtonRight:
			subMenu, err := mc.currentMenu.Execute()
			if err != nil {
				// Could display error, for now just continue
				needsRender = false
			}
			if subMenu != nil {
				mc.currentMenu = subMenu
			}

		case eziog500.ButtonLeft, eziog500.ButtonEsc:
			// Go back to parent menu
			if mc.currentMenu.Parent != nil {
				mc.currentMenu = mc.currentMenu.Parent
			} else {
				// At root menu, exit
				return nil
			}

		default:
			needsRender = false
		}

		if needsRender {
			if err := mc.currentMenu.Render(mc.display); err != nil {
				return err
			}
		}
	}

	return nil
}

// CurrentMenu returns the currently active menu.
func (mc *MenuController) CurrentMenu() *Menu {
	return mc.currentMenu
}

// GoToRoot navigates back to the root menu.
func (mc *MenuController) GoToRoot() {
	mc.currentMenu = mc.rootMenu
}

// Refresh re-renders the current menu.
func (mc *MenuController) Refresh() error {
	return mc.currentMenu.Render(mc.display)
}

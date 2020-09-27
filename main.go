// MIT License Copyright (c) 2020 Hiroshi Shimamoto
// vim: set sw=4 sts=4:
package main

import (
    "fmt"

    "fwdmng/config"
    "github.com/gdamore/tcell"
    "github.com/rivo/tview"
)

type ListItem interface {
    Header(tcell.Screen)
    Print(tcell.Screen, int, bool)
}

type sshhost struct {
    *config.SSHHost
}

func (l *sshhost)Header(screen tcell.Screen) {
    tview.Print(screen, "[yellow::b]Name", 1, 0, 16, tview.AlignLeft, tcell.ColorWhite)
    tview.Print(screen, "[yellow::b]Hostname", 17, 0, 16, tview.AlignLeft, tcell.ColorWhite)
    tview.Print(screen, "[yellow::b]Status", 33, 0, 16, tview.AlignLeft, tcell.ColorWhite)
}

func (l *sshhost)Print(screen tcell.Screen, y int, selected bool) {
    color := tcell.ColorWhite
    if selected { color = tcell.ColorLime }
    tview.Print(screen, l.Name, 1, y, 16, tview.AlignLeft, color)
    tview.Print(screen, l.Hostname, 17, y, 16, tview.AlignLeft, color)
}

type sshfwd struct {
    *config.Fwd
}

func (l *sshfwd)Header(screen tcell.Screen) {
    tview.Print(screen, "[yellow::b]Proto", 2, 0, 16, tview.AlignLeft, tcell.ColorWhite)
    tview.Print(screen, "[yellow::b]Forwarding", 18, 0, 32, tview.AlignLeft, tcell.ColorWhite)
}

func (l *sshfwd)Print(screen tcell.Screen, y int, selected bool) {
    color := tcell.ColorSilver
    if selected { color = tcell.ColorGreen }
    // U+2192 = RIGHTWARDS ARROW
    hostports := fmt.Sprintf("%s \u2192 %s", l.Local, l.Remote)
    tview.Print(screen, l.Name, 2, y, 16, tview.AlignLeft, color)
    tview.Print(screen, hostports, 18, y, 32, tview.AlignLeft, color)
}

type ServiceList struct {
    *tview.Box
    cfg *config.Config
    // ui
    pages *tview.Pages
    items []ListItem
    cursor int
    quit func()
}

func NewServiceList() *ServiceList {
    return &ServiceList{ Box: tview.NewBox() }
}

func (s *ServiceList)UpdateItems() {
    s.items = []ListItem{}
    for i, _ := range s.cfg.SSHHosts {
	host := &s.cfg.SSHHosts[i]
	s.items = append(s.items, &sshhost{ SSHHost: host })
	for j, _ := range host.Fwds {
	    f := &host.Fwds[j]
	    s.items = append(s.items, &sshfwd{ Fwd: f })
	}
    }
}

func (s *ServiceList)Quit() {
    modal := tview.NewModal().
	SetText("Quit?").
	AddButtons([]string{"Quit", "Cancel"}).
	SetDoneFunc(func(idx int, lbl string) {
	    if lbl == "Quit" {
		s.quit()
	    }
	    s.pages.RemovePage("quit")
	})
    s.pages.AddAndSwitchToPage("quit", modal, true)
}

func (s *ServiceList)Draw(screen tcell.Screen) {
    s.UpdateItems()
    s.Box.Draw(screen)
    x, y, w, h := s.GetInnerRect()
    h--
    x++
    for i, item := range s.items {
	y++
	if y >= h {
	    break
	}
	if i == s.cursor {
	    item.Header(screen)
	    item.Print(screen, y, true)
	} else {
	    item.Print(screen, y, false)
	}
    }

    // show footer help
    help := "<Up/Down> Select "
    help += "| <Enter> [::u]E[::-]dit "
    help += "| <Del> [::u]D[::-]elete "
    help += "| [::u]N[::-]ew "
    help += "| [::u]Q[::-]uit"
    tview.Print(screen, help, x, h, w, tview.AlignLeft, tcell.ColorWhite)
}

func (s *ServiceList)InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
    return s.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	last := len(s.items) - 1
	up := func() {
	    s.cursor--
	    if s.cursor < 0 {
		s.cursor = 0
	    }
	}
	down := func() {
	    s.cursor++
	    if s.cursor > last {
		s.cursor = last
	    }
	}
	edit := func() {
	}
	del := func() {
	}
	switch event.Key() {
	case tcell.KeyUp: up()
	case tcell.KeyDown: down()
	case tcell.KeyHome: s.cursor = 0
	case tcell.KeyEnd: s.cursor = last
	case tcell.KeyEnter: edit()
	case tcell.KeyDelete: del()
	}
	switch event.Rune() {
	case 'k': up()
	case 'j': down()
	case 'e': edit()
	case 'd': del()
	case 'n':
	case 'q': s.Quit()
	}
    })
}

func main() {
    fmt.Println("start")
    cfg, err := config.Load("fwdconfig.toml")
    if err != nil {
	fmt.Println(err)
	return
    }
    fmt.Println(*cfg)

    app := tview.NewApplication()

    triblectrlc := 0
    app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyCtrlC {
	    triblectrlc++
	    if triblectrlc >= 3 {
		return event
	    }
	    return nil
	}
	triblectrlc = 0
	return event
    })

    list := NewServiceList()
    list.cfg = cfg

    pages := tview.NewPages()
    pages.AddPage("main", list, true, true)

    list.pages = pages
    list.quit = app.Stop

    app.SetRoot(pages, true)
    if err := app.Run(); err != nil {
	panic(err)
    }

    config.Save(cfg, "fwdconfig.toml")
}

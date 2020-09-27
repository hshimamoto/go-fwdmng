// MIT License Copyright (c) 2020 Hiroshi Shimamoto
// vim: set sw=4 sts=4:
package main

import (
    "fmt"

    "fwdmng/config"
    "github.com/gdamore/tcell"
    "github.com/rivo/tview"
)

type Service struct {
    name, addr, status, stats string
}

type ServiceList struct {
    *tview.Box
    services []*Service
    n int
    quit func()
    edit func(serv *Service)
}

func NewServiceList() *ServiceList {
    return &ServiceList{ Box: tview.NewBox() }
}

func (s *ServiceList)Draw(screen tcell.Screen) {
    s.Box.Draw(screen)
    x, y, w, h := s.GetInnerRect()
    // name, address, status, stats
    labels := []string{"Name", "Address", "Status", "Stats"}
    min := func(a, b int) int {
	if a < b {
	    return a
	}
	return b
    }
    h--
    x++
    w -= 2
    size := []int{
	min(w/4, 16),
	min(w/4, 32),
	min(w/4, 8),
	min(w/4, 80),
    }
    // draw header
    sx := x
    for i, label := range labels {
	sz := size[i]
	txt := "[yellow::b]" + label
	tview.Print(screen, txt, sx, y, sz, tview.AlignLeft, tcell.ColorWhite)
	sx += sz
    }
    y++
    for i, serv := range s.services {
	if y >= h {
	    break
	}
	color := tcell.ColorWhite
	if i == s.n {
	    color = tcell.ColorGreen
	}
	sx = x
	tview.Print(screen, serv.name, sx, y, size[0], tview.AlignLeft, color)
	sx += size[0]
	tview.Print(screen, serv.addr, sx, y, size[1], tview.AlignLeft, color)
	sx += size[1]
	tview.Print(screen, serv.status, sx, y, size[2], tview.AlignLeft, color)
	sx += size[2]
	tview.Print(screen, serv.stats, sx, y, size[3], tview.AlignLeft, color)
	y++
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
	last := len(s.services) - 1
	up := func() {
	    s.n--
	    if s.n < 0 {
		s.n = 0
	    }
	}
	down := func() {
	    s.n++
	    if s.n > last {
		s.n = last
	    }
	}
	edit := func() {
	    s.edit(s.services[s.n])
	}
	del := func() {
	    old := s.services
	    s.services = []*Service{}
	    for i, serv := range old {
		if i != s.n {
		    s.services = append(s.services, serv)
		}
	    }
	    last = len(s.services) - 1
	    if s.n > last {
		s.n = last
	    }
	}
	switch event.Key() {
	case tcell.KeyUp: up()
	case tcell.KeyDown: down()
	case tcell.KeyHome: s.n = 0
	case tcell.KeyEnd: s.n = last
	case tcell.KeyEnter: edit()
	case tcell.KeyDelete: del()
	}
	switch event.Rune() {
	case 'k': up()
	case 'j': down()
	case 'e': edit()
	case 'd': del()
	case 'n':
	    serv := &Service{ "New", "New Address", "disable", "none" }
	    s.services = append(s.services, serv)
	    s.edit(serv)
	case 'q': s.quit()
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
    list.services = []*Service{
	&Service{ "aaaa", "aaaa", "ok", "aaaa" },
	&Service{ "bbbb", "bbbb", "ok", "bbbb" },
	&Service{ "cccc", "cccc", "ok", "cccc" },
    }

    pages := tview.NewPages()
    pages.AddPage("main", list, true, true)

    list.quit = func() {
	modal := tview.NewModal().
	    SetText("Quit?").
	    AddButtons([]string{"Quit", "Cancel"}).
	    SetDoneFunc(func(idx int, lbl string) {
		if lbl == "Quit" {
		    app.Stop()
		}
		pages.RemovePage("quit")
	    })
	pages.AddAndSwitchToPage("quit", modal, true)
    }

    list.edit = func(serv *Service) {
	form := tview.NewForm()
	form.AddInputField("Name", serv.name, 16, nil, nil)
	form.AddInputField("Address", serv.addr, 32, nil, nil)
	namef := form.GetFormItemByLabel("Name")
	name, _ := namef.(*tview.InputField)
	addrf := form.GetFormItemByLabel("Address")
	addr, _ := addrf.(*tview.InputField)
	form.AddButton("Done", func() {
	    serv.name = name.GetText()
	    serv.addr = addr.GetText()
	    pages.RemovePage("edit")
	})
	form.AddButton("Cancel", func() {
	    pages.RemovePage("edit")
	})
	pages.AddAndSwitchToPage("edit", form, true)
    }

    app.SetRoot(pages, true)
    if err := app.Run(); err != nil {
	panic(err)
    }

    config.Save(cfg, "fwdconfig.toml")
}

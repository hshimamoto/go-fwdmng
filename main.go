// MIT License Copyright (c) 2020 Hiroshi Shimamoto
// vim: set sw=4 sts=4:
package main

import (
    "fmt"
    "time"

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
    status string
}

func (l *sshhost)Header(screen tcell.Screen) {
    tview.Print(screen, "[yellow::b]Name", 1, 0, 16, tview.AlignLeft, tcell.ColorWhite)
    tview.Print(screen, "[yellow::b]Hostname", 17, 0, 32, tview.AlignLeft, tcell.ColorWhite)
    tview.Print(screen, "[yellow::b]Status", 49, 0, 16, tview.AlignLeft, tcell.ColorWhite)
}

func (l *sshhost)Print(screen tcell.Screen, y int, selected bool) {
    color := tcell.ColorWhite
    if selected { color = tcell.ColorLime }
    tview.Print(screen, l.Name, 1, y, 16, tview.AlignLeft, color)
    tview.Print(screen, l.Hostname, 17, y, 32, tview.AlignLeft, color)
    tview.Print(screen, l.status, 49, y, 16, tview.AlignLeft, color)
}

func (h *sshhost)Connect(done func()) {
    h.status = "connecting"
    go func() {
	time.Sleep(time.Second * 5)
	h.status = "connected"
	done()
    }()
}

func (h *sshhost)Disconnect(done func()) {
    h.status = "disconnecting"
    go func() {
	time.Sleep(time.Second * 5)
	h.status = "disconnected"
	done()
    }()
}

type sshfwd struct {
    *config.Fwd
    host *sshhost
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
    hosts []*sshhost
    // ui
    app *Application
    items []ListItem
    cursor int
}

func NewServiceList(cfg *config.Config) *ServiceList {
    s := &ServiceList{ Box: tview.NewBox() }
    // copy from config
    s.hosts = []*sshhost{}
    for i, _ := range cfg.SSHHosts {
	h := &cfg.SSHHosts[i]
	s.hosts = append(s.hosts, &sshhost{ SSHHost: h, status: "disconnected" })
    }
    return s
}

func (s *ServiceList)UpdateItems() {
    s.items = []ListItem{}
    for _, host := range s.hosts {
	s.items = append(s.items, host)
	for j, _ := range host.Fwds {
	    f := &host.Fwds[j]
	    s.items = append(s.items, &sshfwd{ Fwd: f, host: host })
	}
    }
}

func (s *ServiceList)Confirm(ask string, f func()) {
    modal := tview.NewModal().
	SetText(ask).
	AddButtons([]string{"No", "Yes"}).
	SetDoneFunc(func(idx int, lbl string) {
	    if lbl == "Yes" {
		f()
	    }
	    s.app.pages.RemovePage("confirm")
	})
    s.app.pages.AddAndSwitchToPage("confirm", modal, true)
}

func (s *ServiceList)Quit() {
    modal := tview.NewModal().
	SetText("Quit?").
	AddButtons([]string{"Quit", "Cancel"}).
	SetDoneFunc(func(idx int, lbl string) {
	    if lbl == "Quit" {
		s.app.Stop()
	    }
	    s.app.pages.RemovePage("quit")
	})
    s.app.pages.AddAndSwitchToPage("quit", modal, true)
}

func (s *ServiceList)EditSSHHost(host *sshhost) {
    form := tview.NewForm()
    form.AddInputField("Name", host.Name, 16, nil, nil)
    form.AddInputField("Hostname", host.Hostname, 32, nil, nil)
    form.AddInputField("User", host.User, 32, nil, nil)
    form.AddInputField("Privkey", host.Privkey, 32, nil, nil)
    form.AddInputField("Proxy", host.Proxy, 32, nil, nil)
    namef := form.GetFormItemByLabel("Name")
    name, _ := namef.(*tview.InputField)
    hostnamef := form.GetFormItemByLabel("Hostname")
    hostname, _ := hostnamef.(*tview.InputField)
    userf := form.GetFormItemByLabel("User")
    user, _ := userf.(*tview.InputField)
    privkeyf := form.GetFormItemByLabel("Privkey")
    privkey, _ := privkeyf.(*tview.InputField)
    proxyf := form.GetFormItemByLabel("Proxy")
    proxy, _ := proxyf.(*tview.InputField)
    form.AddButton("Done", func() {
	host.Name = name.GetText()
	host.Hostname = hostname.GetText()
	host.User = user.GetText()
	host.Privkey = privkey.GetText()
	host.Proxy = proxy.GetText()
	s.app.pages.RemovePage("edit")
    })
    form.AddButton("Cancel", func() {
	s.app.pages.RemovePage("edit")
    })
    s.app.pages.AddAndSwitchToPage("edit", form, true)
}

func (s *ServiceList)EditSSHFwd(fwd *sshfwd) {
    form := tview.NewForm()
    form.AddInputField("Proto", fwd.Name, 16, nil, nil)
    form.AddInputField("Local", fwd.Local, 32, nil, nil)
    form.AddInputField("Remote", fwd.Remote, 32, nil, nil)
    namef := form.GetFormItemByLabel("Proto")
    name, _ := namef.(*tview.InputField)
    localf := form.GetFormItemByLabel("Local")
    local, _ := localf.(*tview.InputField)
    remotef := form.GetFormItemByLabel("Remote")
    remote, _ := remotef.(*tview.InputField)
    form.AddButton("Done", func() {
	fwd.Name = name.GetText()
	fwd.Local = local.GetText()
	fwd.Remote = remote.GetText()
	s.app.pages.RemovePage("edit")
    })
    form.AddButton("Cancel", func() {
	s.app.pages.RemovePage("edit")
    })
    s.app.pages.AddAndSwitchToPage("edit", form, true)
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
    help1 := "<Up/Down> Select "
    help1 += "| <Enter> [::u]E[::-]dit "
    help1 += "| <Del> [::u]D[::-]elete "
    help1 += "| [::u]N[::-]ew host "
    help1 += "| [::u]A[::-]dd fwd "
    help1 += "| [::u]Q[::-]uit"
    help2 := "[::u]S[::-]tart/Stop "
    tview.Print(screen, help1, x, h-1, w, tview.AlignLeft, tcell.ColorWhite)
    tview.Print(screen, help2, x, h, w, tview.AlignLeft, tcell.ColorWhite)
}

func (s *ServiceList)InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
    return s.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	item := s.items[s.cursor]
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
	    switch item := item.(type) {
	    case *sshhost: s.EditSSHHost(item)
	    case *sshfwd: s.EditSSHFwd(item)
	    }
	}
	newhost := func() {
	    host := &sshhost{}
	    host.SSHHost = &config.SSHHost{
		Name: "new name",
		Hostname: "new hostname",
		User: "new user",
		Privkey: "new privkey",
		Proxy: "",
		Fwds: []config.Fwd{
		    config.Fwd{
			Name: "unknown",
			Local: ":0",
			Remote: "127.0.0.1:0",
		    },
		},
	    }
	    host.status = "disconnected"
	    s.hosts = append(s.hosts, host)
	}
	addfwd := func() {
	    fwd := config.Fwd{
		Name: "unknown",
		Local: ":0",
		Remote: "127.0.0.1:0",
	    }
	    if host, ok := item.(*sshhost); ok {
		host.SSHHost.Fwds = append(host.SSHHost.Fwds, fwd)
	    }
	    if f, ok := item.(*sshfwd); ok {
		host := f.host
		host.SSHHost.Fwds = append(host.SSHHost.Fwds, fwd)
	    }
	}
	del := func() {
	    // what is the target item
	    name := ""
	    target := item
	    if f, ok := target.(*sshfwd); ok {
		if len(f.host.Fwds) == 1 {
		    target = f.host
		}
	    }
	    switch it := target.(type) {
	    case *sshhost: name = it.Name
	    case *sshfwd: name = fmt.Sprintf("%s:%s", it.host.Name, it.Name)
	    }
	    // confirm
	    s.Confirm(fmt.Sprintf("Delete %s ?", name), func() {
		if h, ok := target.(*sshhost); ok {
		    newlist := []*sshhost{}
		    for _, p := range s.hosts {
			if h == p {
			    continue
			}
			newlist = append(newlist, p)
		    }
		    s.hosts = newlist
		}
		if f, ok := target.(*sshfwd); ok {
		    h := f.host
		    newlist := []config.Fwd{}
		    for i, _ := range h.Fwds {
			p := &h.Fwds[i]
			if f.Fwd == p {
			    continue
			}
			newlist = append(newlist, *p)
		    }
		    h.Fwds = newlist
		}
		s.UpdateItems()
		last = len(s.items) - 1
		if s.cursor > last {
		    s.cursor = last
		}
	    })
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
	case 'n': newhost()
	case 'a': addfwd()
	case 'd': del()
	case 's':
	    host, ok := item.(*sshhost)
	    if !ok {
		fwd := item.(*sshfwd)
		host = fwd.host
	    }
	    switch host.status {
	    case "connected":
		host.Disconnect(func() {
		    // TODO
		    s.app.Draw()
		})
	    case "disconnected":
		host.Connect(func() {
		    // TODO
		    s.app.Draw()
		})
	    }
	case 'q': s.Quit()
	}
    })
}

type Application struct {
    *tview.Application
    pages *tview.Pages
    s *ServiceList
    cfgpath string
}

func NewApplication(cfgpath string) *Application {
    app := &Application{
	Application: tview.NewApplication(),
	pages: tview.NewPages(),
	cfgpath: cfgpath,
    }
    cfg, err := config.Load(cfgpath)
    if err != nil {
	fmt.Println(err)
	return nil
    }
    list := NewServiceList(cfg)
    app.pages.AddPage("main", list, true, true)
    list.app = app
    app.s = list
    app.SetRoot(app.pages, true)
    return app
}

func (a *Application)Run() error {
    triple_ctrl_c := 0
    a.Application.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyCtrlC {
	    triple_ctrl_c++
	    if triple_ctrl_c >= 3 {
		return event
	    }
	    return nil
	}
	triple_ctrl_c = 0
	return event
    })
    return a.Application.Run()
}

func (a *Application)Stop() {
    // for save sshhosts
    cfg := &config.Config{}
    cfg.SSHHosts = []config.SSHHost{}
    for _, host := range a.s.hosts {
	cfg.SSHHosts = append(cfg.SSHHosts, *host.SSHHost)
    }
    config.Save(cfg, a.cfgpath)
    a.Application.Stop()
}

func main() {
    fmt.Println("start")

    app := NewApplication("fwdconfig.toml")
    if err := app.Run(); err != nil {
	panic(err)
    }
}

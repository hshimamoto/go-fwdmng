// MIT License Copyright (c) 2020 Hiroshi Shimamoto
// vim: set sw=4 sts=4:
package main

import (
    "fmt"
    "io/ioutil"
    "net"
    "strings"
    "time"

    "fwdmng/config"
    "github.com/gdamore/tcell"
    "github.com/rivo/tview"

    "golang.org/x/crypto/ssh"
    "github.com/hshimamoto/go-session"
)

type ListItem interface {
    Header(tcell.Screen)
    Print(tcell.Screen, int, bool)
}

type sshhost struct {
    *config.SSHHost
    status string
    client *ssh.Client
    fwds []*sshfwd
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
	failure := func() {
	    h.status = "failure"
	    done()
	    time.Sleep(time.Second * 5)
	    h.status = "disconnected"
	    done()
	}
	cfg := &ssh.ClientConfig{
	    User: h.User,
	    HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	// get key
	buf, err := ioutil.ReadFile(h.Privkey)
	if err != nil {
	    failure()
	    return
	}
	key, err := ssh.ParsePrivateKey(buf)
	if err != nil {
	    failure()
	    return
	}
	cfg.Auth = []ssh.AuthMethod{ ssh.PublicKeys(key) }
	//
	var conn net.Conn
	hostport := h.Hostname
	if strings.Index(hostport, ":") < 0 {
	    hostport += ":22"
	}
	if h.Proxy != "" {
	    var err error
	    conn, err = session.Corkscrew(h.Proxy, hostport)
	    if err != nil {
		failure()
		return
	    }
	} else {
	    var err error
	    conn, err = session.Dial(hostport)
	    if err != nil {
		failure()
		return
	    }
	}
	// ssh handshake
	cconn, cchans, creqs, err := ssh.NewClientConn(conn, hostport, cfg)
	if err != nil {
	    failure()
	    return
	}
	h.client = ssh.NewClient(cconn, cchans, creqs)
	h.status = "connected"
	// start local servers
	for _, f := range h.fwds {
	    f.LocalStart()
	}
	done()
    }()
}

func (h *sshhost)Disconnect(done func()) {
    h.status = "disconnecting"
    go func() {
	if h.client == nil {
	    time.Sleep(time.Second)
	    h.status = "disconnected"
	    done()
	    return
	}
	// close client
	h.client.Close()
	h.client = nil
	time.Sleep(time.Second)
	// stopping fwds
	for _, f := range h.fwds {
	    go f.LocalStop()
	}
	time.Sleep(time.Second)
	h.status = "disconnected"
	done()
	return
    }()
}

type sshfwd struct {
    *config.Fwd
    host *sshhost
    serv *session.Server
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
    tview.Print(screen, hostports, 20, y, 32, tview.AlignLeft, color)
    if l.serv != nil {
	tview.Print(screen, "listening", 54, y, 16, tview.AlignLeft, color)
    }
}

func (f *sshfwd)LocalStart() {
    if f.serv != nil {
	return
    }
    hp := strings.Split(f.Local, ":")
    if len(hp) != 2 {
	return
    }
    if hp[1] == "0" {
	return
    }
    serv, err := session.NewServer(f.Local, func(conn net.Conn) {
	defer conn.Close()
    })
    if err != nil {
	return
    }
    go serv.Run()
    f.serv = serv
}

func (f *sshfwd)LocalStop() {
    if f.serv != nil {
	f.serv.Stop()
	time.Sleep(time.Second)
	f.serv = nil
    }
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
	host := &sshhost{
	    SSHHost: &cfg.SSHHosts[i],
	    status: "disconnected",
	    client: nil,
	    fwds: []*sshfwd{},
	}
	// copy fwds
	for j, _ := range host.Fwds {
	    fwd := &sshfwd{
		Fwd: &host.Fwds[j],
		host: host,
		serv: nil,
	    }
	    host.fwds = append(host.fwds, fwd)
	}
	s.hosts = append(s.hosts, host)
    }
    return s
}

func (s *ServiceList)UpdateItems() {
    s.items = []ListItem{}
    for _, host := range s.hosts {
	s.items = append(s.items, host)
	for _, fwd := range host.fwds {
	    s.items = append(s.items, fwd)
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
	changed := false
	fwd.Name = name.GetText()
	if fwd.Local != local.GetText() {
	    fwd.Local = local.GetText()
	    changed = true
	}
	fwd.Remote = remote.GetText()
	s.app.pages.RemovePage("edit")
	if fwd.host.status == "connected" {
	    if changed {
		fwd.LocalStop()
	    }
	    fwd.LocalStart()
	}
    })
    form.AddButton("Cancel", func() {
	s.app.pages.RemovePage("edit")
    })
    s.app.pages.AddAndSwitchToPage("edit", form, true)
}

func (s *ServiceList)DeleteItem() {
    target := s.items[s.cursor]
    if fwd, ok := target.(*sshfwd); ok {
	if len(fwd.host.fwds) == 1 {
	    target = fwd.host
	}
    }
    // check running?
    if host, ok := target.(*sshhost); ok {
	if host.status != "disconnected" {
	    // ignore
	    return
	}
    }
    name := "unknown"
    switch it := target.(type) {
    case *sshhost: name = it.Name
    case *sshfwd: name = fmt.Sprintf("%s:%s", it.host.Name, it.Name)
    }
    // confirm
    s.Confirm(fmt.Sprintf("Delete %s ?", name), func() {
	switch it := target.(type) {
	case *sshhost:
	    newlist := []*sshhost{}
	    for _, p := range s.hosts {
		if it != p {
		    newlist = append(newlist, p)
		}
	    }
	    s.hosts = newlist
	case *sshfwd:
	    // stop if it running in background
	    if it.serv != nil {
		go it.LocalStop()
	    }
	    newlist := []*sshfwd{}
	    for _, p := range it.host.fwds {
		if it != p {
		    newlist = append(newlist, p)
		}
	    }
	    it.host.fwds = newlist
	}
    })
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
	    host.client = nil
	    host.fwds = []*sshfwd{
		&sshfwd{
		    Fwd: &config.Fwd{
			Name: "unknown",
			Local: ":0",
			Remote: "127.0.0.1:0",
		    },
		    host: host,
		    serv: nil,
		},
	    }
	    s.hosts = append(s.hosts, host)
	}
	addfwd := func() {
	    fwd := &sshfwd{
		Fwd: &config.Fwd{
		    Name: "unknown",
		    Local: ":0",
		    Remote: "127.0.0.1:0",
		},
	    }
	    host, _ := item.(*sshhost)
	    if f, ok := item.(*sshfwd); ok {
		host = f.host
	    }
	    fwd.host = host
	    host.fwds = append(host.fwds, fwd)
	}
	switch event.Key() {
	case tcell.KeyUp: up()
	case tcell.KeyDown: down()
	case tcell.KeyHome: s.cursor = 0
	case tcell.KeyEnd: s.cursor = last
	case tcell.KeyEnter: edit()
	case tcell.KeyDelete: s.DeleteItem()
	}
	switch event.Rune() {
	case 'k': up()
	case 'j': down()
	case 'e': edit()
	case 'n': newhost()
	case 'a': addfwd()
	case 'd': s.DeleteItem()
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
	// update Fwds
	host.SSHHost.Fwds = []config.Fwd{}
	for _, fwd := range host.fwds {
	    host.SSHHost.Fwds = append(host.SSHHost.Fwds, *fwd.Fwd)
	}
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

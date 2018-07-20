package main

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/fluffle/goirc/logging"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
)

var setLayeredWindowAttributes uintptr

const (
	LWA_COLORKEY = 1
	LWA_ALPHA    = 2
)

func init() {
	libuser32 := win.MustLoadLibrary("user32.dll")
	setLayeredWindowAttributes = win.MustGetProcAddress(libuser32, "SetLayeredWindowAttributes")
}

func SetLayeredWindowAttributes(hwnd win.HWND, crKey, bAlpha, dwFlags int32) bool {
	ret, _, _ := syscall.Syscall6(setLayeredWindowAttributes, 4,
		uintptr(hwnd),
		uintptr(crKey),
		uintptr(bAlpha),
		uintptr(dwFlags),
		0, 0)
	return ret != 0
}

const (
	CHATLOG_DIR     = "./chatlogs/"
	SCREENSHOTS_DIR = "./screenshots/"
	SCRIPTS_DIR     = "./scripts/"
	THEMES_DIR      = "./themes/"

	CONNECT_RETRIES        = 1
	CONNECT_RETRY_INTERVAL = time.Second
	CONNECT_TIMEOUT        = time.Second * 30
)

var (
	mw        *myMainWindow
	tabWidget *walk.TabWidget
	statusBar *walk.StatusBarItem

	mainWindowFocused bool = true // start focused because windows
	mainWindowHidden  bool = false
)

type myMainWindow struct {
	*walk.MainWindow
}

func (mw *myMainWindow) WndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	if msg == win.WM_SYSCOMMAND {
		// minimize/close to tray
		if wParam == win.SC_MINIMIZE || wParam == win.SC_CLOSE {
			win.ShowWindow(mw.Handle(), win.SW_HIDE)
			mainWindowHidden = true
			return 0
		}
	}
	return mw.MainWindow.WndProc(hwnd, msg, wParam, lParam)
}

func main() {
	runtime.LockOSThread()

	logging.SetLogger(&debugLogger{})

	mw = new(myMainWindow)
	MainWindow{
		AssignTo: &mw.MainWindow,
		Title:    "chopsuey IRC " + VERSION_STRING,
		Layout: VBox{
			MarginsZero: true,
			SpacingZero: true,
		},
		Children: []Widget{},
		StatusBarItems: []StatusBarItem{
			StatusBarItem{
				AssignTo: &statusBar,
				Text:     "not connected to any networks...",
			},
		},
	}.Create()
	walk.InitWrapperWindow(mw)

	//
	// transparency:
	//
	// required:
	// win.SetWindowLong(mw.Handle(), win.GWL_EXSTYLE, win.WS_EX_CONTROLPARENT|win.WS_EX_LAYERED)
	//
	// entire window, 50% transparent
	// SetLayeredWindowAttributes(mw.Handle(), 0, 0xff * 50, LWA_ALPHA)
	//
	// chromakey
	// NOTE(tso): no
	// SetLayeredWindowAttributes(mw.Handle(), 0xffffff, 0, LWA_COLORKEY)

	//
	// borderless, attempt #1
	//
	// NOTE(tso): fucks up layout (hides textInput)
	// win.SetWindowLong(mw.Handle(), win.GWL_STYLE, win.WS_OVERLAPPEDWINDOW&(^win.WS_CAPTION))

	var err error
	tabWidget, err = walk.NewTabWidgetWithStyle(mw, win.TCS_MULTILINE)
	checkErr(err)

	mw.Children().Add(tabWidget)

	mw.SetBounds(walk.Rectangle{
		X:      1536,
		Y:      0,
		Width:  384,
		Height: 1050,
	})

	ico, err := walk.NewIconFromFile("chopsuey.ico")
	checkErr(err)
	mw.SetIcon(ico)

	systray, err := walk.NewNotifyIcon()
	checkErr(err)
	defer systray.Dispose()
	systray.SetIcon(ico)
	systray.SetVisible(true)
	systray.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			if mainWindowHidden {
				win.ShowWindow(mw.Handle(), win.SW_NORMAL)
			} else {
				win.ShowWindow(mw.Handle(), win.SW_HIDE)
			}
			mainWindowHidden = !mainWindowHidden
		}
	})
	/*	action := walk.NewAction()
		action.SetText("hello world")
		systray.ContextMenu().Actions().Add(action)
	*/
	font, err := walk.NewFont("ProFontWindows", 9, 0)
	checkErr(err)
	mw.WindowBase.SetFont(font)
	/*
		userTheme, err := loadPaletteFromFile("zenburn")
		checkErr(err)
		loadColorPalette(userTheme[:16])
		bg := userTheme[16]
		r, g, b := byte((bg>>16)&0xff), byte((bg>>8)&0xff), byte(bg&0xff)
		brush, err := walk.NewSolidColorBrush(walk.RGB(r, g, b))
		checkErr(err)
		defer brush.Dispose()
		mw.SetBackground(brush)
		tabWidget.SetBackground(brush)
		sb := mw.StatusBar()
		sb.SetBackground(brush)
		fg := userTheme[17]
		colorref := win.COLORREF(fg&0xff<<16 | fg&0xff00 | fg&0xff0000>>16)
		win.SetTextColor(win.GetDC(mw.Handle()), colorref)
		win.SetTextColor(win.GetDC(tabWidget.Handle()), colorref)
		win.SetTextColor(win.GetDC(sb.Handle()), colorref)

		globalBackgroundColor = bg
		globalForegroundColor = fg
	*/
	tabWidget.SetPersistent(false)

	// NOTE(tso): contrary to what the name of this event publisher implies
	//            CurrentIndexChanged() fires every time you Insert() or Remove()
	//            a TabPage regardless of whether the CurrentIndex() actually
	//            changed.
	//
	//            and you *have* to set the CurrentIndex() again when you Add(),
	//            Insert() or Remove() for everything to draw correctly
	//
	//            e.g.
	//            tabs: [0 1 2 3], currentIndex == 1
	//            Add()
	//            tabs: [0 1 2 3 4], currentIndex still == 1
	//            CurrentIndexChanged fires
	//              uhhhhhhhhhhhhhhhhhhh
	//
	//            at least that's what I think is happening, probably wrong about
	//            something
	// -tso 7/14/2018 11:29:50 PM

	var currentFocusedTab tab
	tabWidget.CurrentIndexChanged().Attach(func() {
		currentTab := clientState.CurrentTab()
		if currentFocusedTab != currentTab {
			currentFocusedTab = currentTab
			currentTab.Focus()
		}
	})
	mw.Activating().Attach(func() {
		mainWindowFocused = true
		// always call Focus() when window regains focus
		clientState.CurrentTab().Focus()
	})
	mw.Deactivating().Attach(func() {
		mainWindowFocused = false
	})
	mw.SizeChanged().Attach(func() {
		for _, t := range clientState.tabs {
			switch t.(type) {
			case *tabServer:
				t := t.(*tabServer)
				t.textBuffer.SendMessage(win.WM_VSCROLL, win.SB_BOTTOM, 0)
			case *tabChannel:
				t := t.(*tabChannel)
				t.Resize()
				t.textBuffer.SendMessage(win.WM_VSCROLL, win.SB_BOTTOM, 0)
			case *tabPrivmsg:
				t := t.(*tabPrivmsg)
				t.textBuffer.SendMessage(win.WM_VSCROLL, win.SB_BOTTOM, 0)
			}
		}
	})

	clientState = &_clientState{
		connections: []*serverConnection{},
		servers:     []*serverState{},
		tabs:        []tab{},
		mu:          &sync.Mutex{},
	}
	clientState.cfg, err = getClientConfig()
	if err != nil {
		log.Println("error parsing config.json", err)
		walk.MsgBox(mw, "error parsing config.json", err.Error(), walk.MsgBoxIconError)
		statusBar.SetText("error parsing config.json")
		/*}

		// XXX TEMPORARY SECRETARY
		for i := 0; i < 1; i++ {
			emptyTab := NewServerTab(&serverConnection{}, &serverState{
				networkName: "tab " + strconv.Itoa(i),
				user:        &userState{nick: "tso"},
			})

			mw.WindowBase.Synchronize(func() {
				paletteCmd(&commandContext{tab: emptyTab})
			})
		}
		*/

	} else {
		for _, cfg := range clientState.cfg.AutoConnect {
			servState := &serverState{
				connState:   CONNECTION_EMPTY,
				hostname:    cfg.Host,
				port:        cfg.Port,
				ssl:         cfg.Ssl,
				networkName: serverAddr(cfg.Host, cfg.Port),
				user: &userState{
					nick: cfg.Nick,
				},
				channels: map[string]*channelState{},
				privmsgs: map[string]*privmsgState{},
			}
			var servConn *serverConnection
			servConn = NewServerConnection(servState,
				func(nickservPASSWORD string, autojoin []string) func() {
					return func() {
						if nickservPASSWORD != "" {
							servConn.conn.Privmsg("NickServ", "IDENTIFY "+nickservPASSWORD)
						}
						for _, channel := range autojoin {
							servConn.conn.Join(channel)
						}
					}
				}(cfg.NickServPASSWORD, cfg.AutoJoin),
			)
			clientState.mu.Lock()
			servView := NewServerTab(servConn, servState)
			clientState.mu.Unlock()
			servState.tab = servView
			servConn.Connect(servState)
		}
	}

	mw.Run()
}

type debugLogger struct{}

func (l *debugLogger) Debug(f string, a ...interface{}) { fmt.Printf(f+"\n", a...) }
func (l *debugLogger) Info(f string, a ...interface{})  { fmt.Printf(f+"\n", a...) }
func (l *debugLogger) Warn(f string, a ...interface{})  { fmt.Printf(f+"\n", a...) }
func (l *debugLogger) Error(f string, a ...interface{}) { fmt.Printf(f+"\n", a...) }

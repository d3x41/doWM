package wm

import (
	"encoding/binary"
	"fmt"
	"log/slog"
	"math"
	"os/exec"
	"strconv"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/keybind"
)

var (
    XUtil *xgbutil.XUtil
)

type Window struct{
    id xproto.Window
    X,Y int
    Width, Height int
    Fullscreen bool
}

type Workspace struct{
    clients map[xproto.Window]xproto.Window
    frametoclient map[xproto.Window]xproto.Window
    windows map[xproto.Window]*Window
}

type WindowManager struct{
    conn *xgb.Conn
    root xproto.Window
    width, height int
    workspaces []Workspace
    workspaceIndex int
    currWorkspace *Workspace
    atoms map[string]xproto.Atom
}


func Create() (*WindowManager, error){
    X, err := xgb.NewConn()
    if err!=nil{
        slog.Error("Couldn't open X display")
        return nil, fmt.Errorf("Couldn't open X display")
    }
    
    // inside Create:
    XUtil, err = xgbutil.NewConnXgb(X)
    if err != nil {
        return nil, fmt.Errorf("couldn't create xgbutil connection: %w", err)
    }

    keybind.Initialize(XUtil)


    setup:=xproto.Setup(X)
    screen:=setup.DefaultScreen(X)

    root := screen.Root

    dimensions,err := xproto.GetGeometry(X, xproto.Drawable(root)).Reply()

    if err != nil{
        return nil, fmt.Errorf("couldn't get screen dimensions: %w", err)
    }

    workspaces := make([]Workspace, 10)
    for i := range workspaces{
        workspaces[i] = Workspace{ 
            clients: map[xproto.Window]xproto.Window{},
            frametoclient: map[xproto.Window]xproto.Window{},
            windows: map[xproto.Window]*Window{},
        }
    }

    return &WindowManager{
        conn: X,
        root: root,
        width: int(dimensions.Width),
        height: int(dimensions.Height),
        workspaces: workspaces,
        currWorkspace: &workspaces[0],
        workspaceIndex: 0,
        atoms: map[string]xproto.Atom{},
    }, nil
}

func (wm *WindowManager) Run(){
    fmt.Println("window manager up and running")

    err := xproto.ChangeWindowAttributesChecked(
        wm.conn, 
        wm.root,
        xproto.CwEventMask,
        []uint32{
            xproto.EventMaskSubstructureNotify |
            xproto.EventMaskSubstructureRedirect,
        },
    ).Check()

    if err!=nil{
        if err.Error()=="BadAccess"{
            slog.Error("other window manager running on display")
            return
        }
    }

    wm.broadcastWorkspace(0)
    wm.broadcastWorkspaceCount()

    err = xproto.GrabServerChecked(
        wm.conn,
    ).Check()

    if err != nil{
        slog.Error("Couldn't grab X server", "error:", err)
        return
    }

    tree, err := xproto.QueryTree(
        wm.conn,
        wm.root,
    ).Reply()

    if err != nil{
        slog.Error("Couldn't query tree", "error:", err)
        return
    }

    root, TopLevelWindows := tree.Root, tree.Children

    if(root != wm.root){
        slog.Error("tree root not equal to window manager root", "error:", err.Error())
        return
    }

    for _, window := range TopLevelWindows{
        if !shouldIgnoreWindow(wm.conn, window){
            wm.Frame(window, true)
        }
    }

    err = xproto.UngrabServerChecked(wm.conn).Check()

    if err!=nil{
        slog.Error("couldn't ungrab server", "error:", err.Error())
        return
    }

    cKeyCode := keybind.StrToKeycodes(XUtil, "c")[0]
    fKeyCode := keybind.StrToKeycodes(XUtil, "f")[0]
    wKeyCode := keybind.StrToKeycodes(XUtil, "w")[0]
    f1KeyCode := keybind.StrToKeycodes(XUtil, "f1")[0]
    f2KeyCode := keybind.StrToKeycodes(XUtil, "f2")[0]
    f3KeyCode := keybind.StrToKeycodes(XUtil, "f3")[0]
    oneKeyCode := keybind.StrToKeycodes(XUtil, "1")[0]
    twoKeyCode := keybind.StrToKeycodes(XUtil, "2")[0]
    threeKeyCode := keybind.StrToKeycodes(XUtil, "3")[0]
    fourKeyCode := keybind.StrToKeycodes(XUtil, "4")[0]
    fiveKeyCode := keybind.StrToKeycodes(XUtil, "5")[0]
    sixKeyCode := keybind.StrToKeycodes(XUtil, "6")[0]
    sevenKeyCode := keybind.StrToKeycodes(XUtil, "7")[0]
    eightKeyCode := keybind.StrToKeycodes(XUtil, "8")[0]
    nineKeyCode := keybind.StrToKeycodes(XUtil, "9")[0]
    zeroKeyCode := keybind.StrToKeycodes(XUtil, "0")[0]

    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1|xproto.ModMaskShift, cKeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1, fKeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1, cKeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1, wKeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1, f1KeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1, f2KeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1, f3KeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1, oneKeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1, twoKeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1, threeKeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1, fourKeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1, fiveKeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1, sixKeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1, sevenKeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1, eightKeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1, nineKeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1, zeroKeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1|xproto.ModMaskShift, oneKeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1|xproto.ModMaskShift, twoKeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1|xproto.ModMaskShift, threeKeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1|xproto.ModMaskShift, fourKeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1|xproto.ModMaskShift, fiveKeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1|xproto.ModMaskShift, sixKeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1|xproto.ModMaskShift, sevenKeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1|xproto.ModMaskShift, eightKeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1|xproto.ModMaskShift, nineKeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1|xproto.ModMaskShift, zeroKeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()

    err = xproto.GrabButtonChecked(wm.conn, true, wm.root, 	uint16(xproto.EventMaskButtonPress | xproto.EventMaskButtonRelease | xproto.EventMaskPointerMotion), xproto.GrabModeAsync, xproto.GrabModeAsync, xproto.WindowNone, xproto.AtomNone, xproto.ButtonIndex1, xproto.ModMask1).Check()

    err = xproto.GrabButtonChecked(wm.conn, true, wm.root, 	uint16(xproto.EventMaskButtonPress | xproto.EventMaskButtonRelease | xproto.EventMaskPointerMotion), xproto.GrabModeAsync, xproto.GrabModeAsync, xproto.WindowNone, xproto.AtomNone, xproto.ButtonIndex3, xproto.ModMask1).Check()

    if err!=nil{
        slog.Error("couldn't grab window+c key", "error:", err.Error())
        return
    }

    var start xproto.ButtonPressEvent
    var attr *xproto.GetGeometryReply


    atoms := []string{
        "_NET_WM_STATE",
        "_NET_WM_STATE_FULLSCREEN",
        "_NET_WM_STATE_ABOVE",
        "_NET_WM_STATE_BELOW",
        "_NET_WM_STATE_MAXIMIZED_HORZ",
        "_NET_WM_STATE_MAXIMIZED_VERT",
    }

    for _, name := range atoms {
        a, _ := xproto.InternAtom(wm.conn, false, uint16(len(name)), name).Reply()
        fmt.Printf("%s = %d\n", name, a.Atom)
        wm.atoms[name] = a.Atom
    }
    for{
        event, err := wm.conn.PollForEvent()
        if err!=nil{
            slog.Error("event error","error:" , err.Error())
            continue
        }
        if event == nil{
            continue
        }
        switch event.(type){
            case xproto.ButtonPressEvent:
                ev := event.(xproto.ButtonPressEvent)
                if ev.Child!=0{
                    attr, _ = xproto.GetGeometry(wm.conn, xproto.Drawable(ev.Child)).Reply()
                    start = ev
                    if ev.Detail == xproto.ButtonIndex1{ 
                        xproto.ConfigureWindow(
                            wm.conn,
                            ev.Child,
                            xproto.ConfigWindowStackMode,
                            []uint32{xproto.StackModeAbove},
                        )
                    }
                }
            case xproto.ButtonReleaseEvent:
                start.Child = 0
            case xproto.MotionNotifyEvent:
                ev := event.(xproto.MotionNotifyEvent)
                if start.Child != 0&&ev.State&xproto.ModMask1!=0{
                    if wm.currWorkspace.windows[start.Child]!=nil&&wm.currWorkspace.windows[start.Child].Fullscreen{
                        break
                    }
                    xdiff := ev.RootX - start.RootX
                    ydiff := ev.RootY - start.RootY
                    Xoffset := attr.X + xdiff
                    Yoffset := attr.Y + ydiff
                    sizeY := attr.Height
                    sizeX := attr.Width
                    fmt.Println("start detail")
                    fmt.Println(start.Detail)
                    if start.Detail == xproto.ButtonIndex3{
                        Xoffset = attr.X
                        Yoffset = attr.Y
                        sizeX = uint16(max(10, int(int16(attr.Width)+xdiff)))
                        sizeY = uint16(max(10, int(int16(attr.Height)+ydiff)))
                    }

                    xproto.ConfigureWindow(
                        wm.conn,
                        start.Child,
                        xproto.ConfigWindowX | xproto.ConfigWindowY |
                        xproto.ConfigWindowWidth | xproto.ConfigWindowHeight,
                        []uint32{uint32(Xoffset), uint32(Yoffset), uint32(sizeX), uint32(sizeY)},
                    )
                    if(sizeX!=attr.Width){
                        client:=wm.currWorkspace.frametoclient[start.Child]
                        xproto.ConfigureWindow(
                            wm.conn,
                            client,
                            xproto.ConfigWindowWidth | xproto.ConfigWindowHeight,
                            []uint32{
                                uint32(sizeX),
                                uint32(sizeY),
                            },
                        )
                    }
                }
            case xproto.CreateNotifyEvent:
                fmt.Println("create notify")
                break
            case xproto.ConfigureRequestEvent:
                wm.OnConfigureRequest(event.(xproto.ConfigureRequestEvent))
                break
            case xproto.MapRequestEvent:
                fmt.Println("MapRequest")
                wm.OnMapRequest(event.(xproto.MapRequestEvent))
                break
            case xproto.ReparentNotifyEvent:
                fmt.Println("reparent notify")
                break
            case xproto.MapNotifyEvent:
                fmt.Println("MapNotify")
                break
            case xproto.ConfigureNotifyEvent:
                fmt.Println("ConfigureNotify")
                break
            case xproto.UnmapNotifyEvent:
                fmt.Println("unmapping")
                wm.OnUnmapNotify(event.(xproto.UnmapNotifyEvent))
                break
            case xproto.DestroyNotifyEvent:
                fmt.Println("DestroyNotify")
                ev:=event.(xproto.DestroyNotifyEvent)
                fmt.Println("Window:")
                fmt.Println(ev.Window)
                fmt.Println("Event:")
                fmt.Println(ev.Event)
                if _, ok := wm.currWorkspace.clients[ev.Window]; ok{
                    delete(wm.currWorkspace.frametoclient, wm.currWorkspace.clients[ev.Window])
                    delete(wm.currWorkspace.windows, wm.currWorkspace.clients[ev.Window])
                    delete(wm.currWorkspace.clients, ev.Window)
                }
                break
            case xproto.EnterNotifyEvent:
                fmt.Println("EnterNotify")
                ev:=event.(xproto.EnterNotifyEvent)
                fmt.Println(ev.Event)
                wm.OnEnterNotify(event.(xproto.EnterNotifyEvent))
                break
            case xproto.LeaveNotifyEvent:
                fmt.Println("LeaveNotify")
                ev:=event.(xproto.LeaveNotifyEvent)
                fmt.Println(ev.Event)
                wm.OnLeaveNotify(event.(xproto.LeaveNotifyEvent))
                break
            case xproto.KeyPressEvent:
                fmt.Println("keyPress")
                ev := event.(xproto.KeyPressEvent)
                if ev.State&xproto.ModMask1!=0{
                    if ev.Detail==cKeyCode{
                        SendWmDelete(wm.conn, wm.currWorkspace.frametoclient[ev.Child])
                        fmt.Println(wm.currWorkspace.frametoclient[ev.Child])
                        wm.UnFrame(wm.currWorkspace.frametoclient[ev.Child], false)

                    }else if ev.Detail == cKeyCode && ev.State&(xproto.ModMask1|xproto.ModMaskShift) == (xproto.ModMask1|xproto.ModMaskShift) {
                        // Mod + Shift + C → force close
                        err := xproto.DestroyWindowChecked(wm.conn, wm.currWorkspace.frametoclient[ev.Child]).Check()
                        if err != nil {
                            fmt.Println("Couldn't force destroy:", err)
                        }
                    }else if ev.Detail == fKeyCode{
                        wm.toggleFullScreen(ev.Child)
                    }else if ev.Detail == wKeyCode{
                        err := exec.Command("launcher_t2").Start()
                        if err != nil{
                            slog.Error("couldn't run rofi", "error:", err)
                        }
                    }else if ev.Detail == f1KeyCode{
                        err := exec.Command("pactl", "set-sink-mute", "@DEFAULT_SINK@", "toggle").Start()
                        if err != nil{
                            slog.Error("couldn't toggle mute", "error:", err)
                        }   
                    }else if ev.Detail == f2KeyCode{
                        err := exec.Command("pactl", "set-sink-volume", "@DEFAULT_SINK@", "-5%").Start()
                        if err != nil{
                            slog.Error("couldn't decrease volume", "error:", err)
                        }
                    }else if ev.Detail == f3KeyCode{
                        err := exec.Command("pactl", "set-sink-volume", "@DEFAULT_SINK@", "+5%").Start()
                        if err != nil{
                            slog.Error("couldn't increase volume", "error:", err)
                        }
                    }

                    if ev.State&(xproto.ModMask1|xproto.ModMaskShift) == (xproto.ModMask1|xproto.ModMaskShift)&&ev.Child!=wm.root{
                        fmt.Println("moving window")
                        w := ev.Child
                        xproto.ConfigureWindow(
                            wm.conn,
                            w,
                            xproto.ConfigWindowStackMode,
                            []uint32{xproto.StackModeAbove},
                        )
                        switch ev.Detail{
                        case oneKeyCode:
                            client := wm.currWorkspace.frametoclient[w]
                            delete(wm.currWorkspace.clients, wm.currWorkspace.frametoclient[w])
                            delete(wm.currWorkspace.frametoclient, w)
                            wm.switchWorkspace(0)
                            wm.currWorkspace.frametoclient[w]=client
                            wm.currWorkspace.clients[client]=w 
                        case twoKeyCode:
                            client := wm.currWorkspace.frametoclient[w]
                            delete(wm.currWorkspace.clients, wm.currWorkspace.frametoclient[w])
                            delete(wm.currWorkspace.frametoclient, w)
                            wm.switchWorkspace(1)
                            wm.currWorkspace.frametoclient[w]=client
                            wm.currWorkspace.clients[client]=w 
                        case threeKeyCode:
                            client := wm.currWorkspace.frametoclient[w]
                            delete(wm.currWorkspace.clients, wm.currWorkspace.frametoclient[w])
                            delete(wm.currWorkspace.frametoclient, w)
                            wm.switchWorkspace(2)
                            wm.currWorkspace.frametoclient[w]=client
                            wm.currWorkspace.clients[client]=w 
                        case fourKeyCode:
                            client := wm.currWorkspace.frametoclient[w]
                            delete(wm.currWorkspace.clients, wm.currWorkspace.frametoclient[w])
                            delete(wm.currWorkspace.frametoclient, w)
                            wm.switchWorkspace(3)
                            wm.currWorkspace.frametoclient[w]=client
                            wm.currWorkspace.clients[client]=w 
                        case fiveKeyCode:
                            client := wm.currWorkspace.frametoclient[w]
                            delete(wm.currWorkspace.clients, wm.currWorkspace.frametoclient[w])
                            delete(wm.currWorkspace.frametoclient, w)
                            wm.switchWorkspace(4)
                            wm.currWorkspace.frametoclient[w]=client
                            wm.currWorkspace.clients[client]=w
                        case sixKeyCode:
                            client := wm.currWorkspace.frametoclient[w]
                            delete(wm.currWorkspace.clients, wm.currWorkspace.frametoclient[w])
                            delete(wm.currWorkspace.frametoclient, w)
                            wm.switchWorkspace(5)
                            wm.currWorkspace.frametoclient[w]=client
                            wm.currWorkspace.clients[client]=w 
                        case sevenKeyCode:
                            client := wm.currWorkspace.frametoclient[w]
                            delete(wm.currWorkspace.clients, wm.currWorkspace.frametoclient[w])
                            delete(wm.currWorkspace.frametoclient, w)
                            wm.switchWorkspace(6)
                            wm.currWorkspace.frametoclient[w]=client
                            wm.currWorkspace.clients[client]=w 
                        case eightKeyCode:
                            client := wm.currWorkspace.frametoclient[w]
                            delete(wm.currWorkspace.clients, wm.currWorkspace.frametoclient[w])
                            delete(wm.currWorkspace.frametoclient, w)
                            wm.switchWorkspace(7)
                            wm.currWorkspace.frametoclient[w]=client
                            wm.currWorkspace.clients[client]=w 
                        case nineKeyCode:
                            client := wm.currWorkspace.frametoclient[w]
                            delete(wm.currWorkspace.clients, wm.currWorkspace.frametoclient[w])
                            delete(wm.currWorkspace.frametoclient, w)
                            wm.switchWorkspace(8)
                            wm.currWorkspace.frametoclient[w]=client
                            wm.currWorkspace.clients[client]=w 
                        case zeroKeyCode:
                            client := wm.currWorkspace.frametoclient[w]
                            delete(wm.currWorkspace.clients, wm.currWorkspace.frametoclient[w])
                            delete(wm.currWorkspace.frametoclient, w)
                            wm.switchWorkspace(9)
                            wm.currWorkspace.frametoclient[w]=client
                            wm.currWorkspace.clients[client]=w 
                        }     

                    }else{ 
                        switch ev.Detail{
                        case oneKeyCode:
                            wm.switchWorkspace(0)
                        case twoKeyCode:
                            wm.switchWorkspace(1)
                        case threeKeyCode:
                            wm.switchWorkspace(2)
                        case fourKeyCode:
                            wm.switchWorkspace(3)
                        case fiveKeyCode:
                            wm.switchWorkspace(4)
                        case sixKeyCode:
                            wm.switchWorkspace(5)
                        case sevenKeyCode:
                            wm.switchWorkspace(6)
                        case eightKeyCode:
                            wm.switchWorkspace(7)
                        case nineKeyCode:
                            wm.switchWorkspace(8)
                        case zeroKeyCode:
                            wm.switchWorkspace(9)
                        }
                    }
                }
                break
            case xproto.ClientMessageEvent:
                ev := event.(xproto.ClientMessageEvent)
                atomName, _ := xproto.GetAtomName(wm.conn, xproto.Atom(ev.Type)).Reply()
                fmt.Println("Atom name is:", atomName.Name)
                break
            default:
                fmt.Println("event: "+event.String())
                fmt.Println(event.Bytes())

        }
    }
}
func (wm *WindowManager) toggleFullScreen(Child xproto.Window){
    win := wm.currWorkspace.windows[Child]
    if win != nil{
        if win.Fullscreen{
            win.Fullscreen = false
            err := xproto.ConfigureWindowChecked(
                wm.conn,
                Child,
                xproto.ConfigWindowX | xproto.ConfigWindowY |
                xproto.ConfigWindowWidth | xproto.ConfigWindowHeight | xproto.ConfigWindowBorderWidth,
                []uint32{uint32(win.X), uint32(win.Y), uint32(win.Width), uint32(win.Height), 3},
            ).Check()
            err = xproto.ConfigureWindowChecked(
                wm.conn,
                wm.currWorkspace.frametoclient[Child],
                xproto.ConfigWindowX | xproto.ConfigWindowY |
                xproto.ConfigWindowWidth | xproto.ConfigWindowHeight,
                []uint32{0, 0, uint32(win.Width), uint32(win.Height)},
            ).Check()
            if err != nil{
                slog.Error("couldn't un fullscreen window", "error: ", err)
            }
        }else{
            win.Fullscreen = true
            xproto.ConfigureWindow(wm.conn, Child, xproto.ConfigWindowStackMode, []uint32{xproto.StackModeAbove})
            attr, _ := xproto.GetGeometry(wm.conn, xproto.Drawable(Child)).Reply()
            win := wm.currWorkspace.windows[Child]
            win.X = int(attr.X)
            win.Y = int(attr.Y)
            win.Width = int(attr.Width)
            win.Height = int(attr.Height)
            err := xproto.ConfigureWindowChecked(
                wm.conn,
                Child,
                xproto.ConfigWindowX | xproto.ConfigWindowY |
                xproto.ConfigWindowWidth | xproto.ConfigWindowHeight|xproto.ConfigWindowBorderWidth,
                []uint32{0, 0, uint32(wm.width), uint32(wm.height),  0},
            ).Check()
            err = xproto.ConfigureWindowChecked(
                wm.conn,
                wm.currWorkspace.frametoclient[Child],
                xproto.ConfigWindowX | xproto.ConfigWindowY |
                xproto.ConfigWindowWidth | xproto.ConfigWindowHeight,
                []uint32{0, 0, uint32(wm.width), uint32(wm.height)},
            ).Check()
            if err != nil{
                slog.Error("couldn't fullscreen window", "error:", err)
            }
        }
    }
}

func (wm *WindowManager) broadcastWorkspaceCount() {
    count:=wm.workspaceIndex+1
    otherCount:=0
    for i, workspace := range wm.workspaces{
        if len(workspace.frametoclient)>0{
            otherCount=i
        }
    }
    otherCount+=1
    if otherCount>count{
        count = otherCount
    }
    data := make([]byte, 4)
    binary.LittleEndian.PutUint32(data, uint32(count))

    netNumberAtom, _ := xproto.InternAtom(wm.conn, true, uint16(len("_NET_NUMBER_OF_DESKTOPS")), "_NET_NUMBER_OF_DESKTOPS").Reply()
    cardinalAtom, _ := xproto.InternAtom(wm.conn, true, uint16(len("CARDINAL")), "CARDINAL").Reply()

    xproto.ChangePropertyChecked(
        wm.conn,
        xproto.PropModeReplace,
        wm.root,
        netNumberAtom.Atom,
        cardinalAtom.Atom,
        32,
        1,
        data,
    ).Check()
}

func (wm *WindowManager) broadcastWorkspace(num int){

    data := make([]byte, 4)
    binary.LittleEndian.PutUint32(data, uint32(num))


    netCurrentDesktopAtom, err := xproto.InternAtom(wm.conn, false, uint16(len("_NET_CURRENT_DESKTOP")), "_NET_CURRENT_DESKTOP").Reply()

    if err != nil {
        slog.Error("intern _NET_CURRENT_DESKTOP", "error:", err)
        return
    }

    cardinalAtom, err := xproto.InternAtom(wm.conn, true, uint16(len("CARDINAL")), "CARDINAL").Reply()
    if err != nil {
        slog.Error("intern CARDINAL","error:" ,err)
        return
    }
    fmt.Println(netCurrentDesktopAtom.Atom)
    fmt.Println(cardinalAtom.Atom)
    err = xproto.ChangePropertyChecked(
        wm.conn,
        xproto.PropModeReplace,
        wm.root,
        netCurrentDesktopAtom.Atom, // must not be 0
        cardinalAtom.Atom,          // must not be 0
        32,
        1,
        data,
    ).Check()

    if err != nil{
        slog.Error("couldn't set _NET_CURRENT_DESKTOP", "error:", err)
    }

    wm.broadcastWorkspaceCount()
}

func (wm *WindowManager) switchWorkspace(workspace int){
    if workspace==wm.workspaceIndex{
        return
    }

    for frame := range wm.currWorkspace.frametoclient{
        err := xproto.UnmapWindowChecked(wm.conn, frame).Check()
        if err != nil{
            slog.Error("couldn't unmap window", "error:", err)
            return
        }
    }

    wm.currWorkspace = &wm.workspaces[workspace]
    wm.workspaceIndex = workspace

    for frame := range wm.currWorkspace.frametoclient{
        err := xproto.MapWindowChecked(wm.conn, frame).Check()
        if err != nil{
            slog.Error("couldn't map window", "error:", err)
            return
        }
    }

    wm.broadcastWorkspace(workspace)
}

func SendWmDelete(conn *xgb.Conn, window xproto.Window) error {
    wmProtocolsAtom, _ := xproto.InternAtom(conn, true, uint16(len("WM_PROTOCOLS")), "WM_PROTOCOLS").Reply()
    wmDeleteAtom, _ := xproto.InternAtom(conn, true, uint16(len("WM_DELETE_WINDOW")), "WM_DELETE_WINDOW").Reply()

    prop, err := xproto.GetProperty(conn, false, window, wmProtocolsAtom.Atom, xproto.AtomAtom, 0, (1<<32)-1).Reply()
    if err != nil || prop.Format != 32 {
        return fmt.Errorf("couldn't get WM_PROTOCOLS")
    }

    supportsDelete := false
    for i := 0; i < int(prop.ValueLen); i++ {
        atom := xgb.Get32(prop.Value[i*4:])
        if xproto.Atom(atom) == wmDeleteAtom.Atom {
            supportsDelete = true
            break
        }
    }

    if !supportsDelete {
        return fmt.Errorf("WM_DELETE_WINDOW not supported")
    }

    ev := xproto.ClientMessageEvent{
        Format: 32,
        Window: window,
        Type:   wmProtocolsAtom.Atom,
        Data: xproto.ClientMessageDataUnionData32New(
            []uint32{
                uint32(wmDeleteAtom.Atom),
                uint32(xproto.TimeCurrentTime),
                0, 0, 0,
            },
        ),
    }

    return xproto.SendEventChecked(
        conn,
        false,
        window,
        xproto.EventMaskNoEvent,
        string(ev.Bytes()),
    ).Check()
}

func (wm *WindowManager) OnLeaveNotify(event xproto.LeaveNotifyEvent){
    const BorderWidth = 3
    const Col = 0x8bd5ca

    err := xproto.ChangeWindowAttributesChecked(
        wm.conn,
        event.Event,
        xproto.CwBackPixel|xproto.CwBorderPixel,
        []uint32{
            Col, // background (BG_COLOR)
            Col, // border color (BORDER_COLOR)
        },
    ).Check()
    if err!=nil{
        slog.Error("couldn't remove focus from window", "error:", err)
    }
}

func (wm *WindowManager) OnEnterNotify(event xproto.EnterNotifyEvent){
    err:=xproto.SetInputFocusChecked(wm.conn, xproto.InputFocusPointerRoot, event.Event, xproto.TimeCurrentTime).Check()
    const BorderWidth = 3
    const Col = 0xa6da95
    err = xproto.ChangeWindowAttributesChecked(
        wm.conn,
        event.Event,
        xproto.CwBackPixel|xproto.CwBorderPixel,
        []uint32{
            Col, // background (BG_COLOR)
            Col, // border color (BORDER_COLOR)
        },
    ).Check()
    if err!=nil{
        slog.Error("couldn't set focus on window", "error:", err)
    }
}

func (wm *WindowManager) findWindow(window xproto.Window) (bool,int, xproto.Window){
    for i, workspace := range wm.workspaces{
        if i == wm.workspaceIndex{
            continue
        }

        for frame := range workspace.frametoclient{
            if frame==window{
                return true, i, frame
            }

        }
    }
    return false, 0, 0
}

func (wm *WindowManager) OnUnmapNotify(event xproto.UnmapNotifyEvent){
    if _, ok := wm.currWorkspace.clients[event.Window]; !ok{
        ok, index, frame := wm.findWindow(event.Event)
        if !ok{
            slog.Info("couldn't unmap since window wasn't in clients")
            fmt.Println(event.Window)
            fmt.Println(wm.currWorkspace.clients)
            return
        }else{
            wm.currWorkspace = &wm.workspaces[index]
            client := wm.currWorkspace.frametoclient[frame]
            delete(wm.currWorkspace.clients, wm.currWorkspace.frametoclient[frame])
            delete(wm.currWorkspace.frametoclient, frame)
            wm.workspaces[wm.workspaceIndex].frametoclient[frame]=client
            wm.workspaces[wm.workspaceIndex].clients[client]=frame 
            fmt.Println("frame")
            fmt.Println(frame)
            fmt.Println("index")
            fmt.Println(index)
            wm.currWorkspace = &wm.workspaces[wm.workspaceIndex]
            wm.UnFrame(wm.currWorkspace.frametoclient[frame], true)
            return
        }
    }

    if(event.Event == wm.root){
        slog.Info("Ignore UnmapNotify for reparented pre-existing window")
        fmt.Println(event.Window)
        return
    }

    wm.UnFrame(event.Window, false)
}

func (wm *WindowManager) UnFrame(w xproto.Window, unmapped bool){
    frame := wm.currWorkspace.clients[w]

    if(!unmapped){
        err := xproto.UnmapWindowChecked(
            wm.conn,
            frame,
        ).Check()

        if err!=nil{
            slog.Error("couldn't unmap frame", "error:", err.Error())
            return
        }
    }

    err := xproto.ReparentWindowChecked(
        wm.conn, 
        w,
        wm.root,
        0, 0,
    ).Check()
    if err!=nil{
        slog.Error("couldn't remap window during unmapping", "error:", err.Error())
    }

    err = xproto.ChangeSaveSetChecked(
        wm.conn,
        xproto.SetModeDelete,
        w,
    ).Check()
    
    if err!=nil{
        slog.Error("couldn't remove window from save", "error:", err.Error())
        return
    }

    err = xproto.DestroyWindowChecked(
        wm.conn,
        frame,
    ).Check()

    if err!=nil{
        slog.Error("couldn't destroy frame", "error:", err.Error())
        return
    }

    delete(wm.currWorkspace.clients, w)
    delete(wm.currWorkspace.windows, frame)
    delete(wm.currWorkspace.frametoclient, frame)
    slog.Info("Unmapped", "frame", frame, "window", w)
}



func shouldIgnoreWindow(conn *xgb.Conn, win xproto.Window) bool {
    // Intern the _NET_WM_WINDOW_TYPE atom
    typeAtom, err := xproto.InternAtom(conn, false, uint16(len("_NET_WM_WINDOW_TYPE")), "_NET_WM_WINDOW_TYPE").Reply()
    if err != nil {
        slog.Error("Error getting _NET_WM_WINDOW_TYPE atom", "error", err)
        return false
    }

    // Get the _NET_WM_WINDOW_TYPE property for the window
    actualType, err := xproto.GetProperty(conn, false, win, typeAtom.Atom, xproto.AtomAtom, 0, 1).Reply()
    if err != nil {
        slog.Error("Error getting _NET_WM_WINDOW_TYPE property", "error", err)
        return false
    }

    if len(actualType.Value) == 0 {
        return false
    }

    // Check if the window has the _NET_WM_WINDOW_TYPE_SPLASH, _NET_WM_WINDOW_TYPE_DIALOG, _NET_WM_WINDOW_TYPE_NOTIFICATION, or _NET_WM_WINDOW_TYPE_DOCK
    netWmSplash, err := xproto.InternAtom(conn, false, uint16(len("_NET_WM_WINDOW_TYPE_SPLASH")), "_NET_WM_WINDOW_TYPE_SPLASH").Reply()
    if err != nil {
        slog.Error("Error getting _NET_WM_WINDOW_TYPE_SPLASH atom", "error", err)
        return false
    }

    netWmDialog, err := xproto.InternAtom(conn, false, uint16(len("_NET_WM_WINDOW_TYPE_DIALOG")), "_NET_WM_WINDOW_TYPE_DIALOG").Reply()
    if err != nil {
        slog.Error("Error getting _NET_WM_WINDOW_TYPE_DIALOG atom", "error", err)
        return false
    }

    netWmNotification, err := xproto.InternAtom(conn, false, uint16(len("_NET_WM_WINDOW_TYPE_NOTIFICATION")), "_NET_WM_WINDOW_TYPE_NOTIFICATION").Reply()
    if err != nil {
        slog.Error("Error getting _NET_WM_WINDOW_TYPE_NOTIFICATION atom", "error", err)
        return false
    }

    netWmDock, err := xproto.InternAtom(conn, false, uint16(len("_NET_WM_WINDOW_TYPE_DOCK")), "_NET_WM_WINDOW_TYPE_DOCK").Reply()
    if err != nil {
        slog.Error("Error getting _NET_WM_WINDOW_TYPE_DOCK atom", "error", err)
        return false
    }

    // Check if the window type matches any of the "ignore" types
    windowType := xproto.Atom(binary.LittleEndian.Uint32(actualType.Value))

    if windowType == netWmSplash.Atom || windowType == netWmDialog.Atom || windowType == netWmNotification.Atom || windowType == netWmDock.Atom {
        return true
    }

    return false
}


func (wm *WindowManager) OnMapRequest(event xproto.MapRequestEvent){
    if shouldIgnoreWindow(wm.conn, event.Window){
        fmt.Println("ignored window since it is either dock, splash, dialog or notify")
        err := xproto.MapWindowChecked(
            wm.conn,
            event.Window,
        ).Check()
        if err != nil {
            slog.Error("Couldn't create new window id","error:", err.Error())
        }
        return
    } 

    wm.Frame(event.Window, false)
    err := xproto.MapWindowChecked(
        wm.conn,
        event.Window,
    ).Check()

    xproto.ChangeWindowAttributes(wm.conn, event.Window, xproto.CwBackPixmap, []uint32{xproto.BackPixmapNone})

    if err != nil {
        slog.Error("Couldn't create new window id","error:", err.Error())
    }
}

func (wm *WindowManager) Frame(w xproto.Window, createdBeforeWM bool){

    if _, exists := wm.currWorkspace.clients[w]; exists {
        fmt.Println("Already framed", w)
        return
    }
    const BorderWidth = 3
    //const BorderCol = 0xb7bdf8
    const Col = 0x8bd5ca

    geometry, err :=xproto.GetGeometry(wm.conn, xproto.Drawable(w)).Reply()

    if err!=nil{
        slog.Error("Couldn't get window geometry","error:", err.Error())
        return
    }

    attribs, err := xproto.GetWindowAttributes(
        wm.conn,
        w,
    ).Reply()

    if err!=nil{
        slog.Error("Couldn't get window attributes","error:", err.Error())
        return
    }


    if attribs.OverrideRedirect {
        fmt.Println("Skipping override-redirect window", w)
        return
    }

    if createdBeforeWM && attribs.MapState != xproto.MapStateViewable {
        fmt.Println("Skipping unmapped pre-existing window", w)
        return
    }



    frameId, err := xproto.NewWindowId(wm.conn)
    if err != nil {
        slog.Error("Couldn't create new window id","error:", err.Error())
        return
    }

    windowMidX:=math.Round(float64(geometry.Width)/2)
    windowMidY:=math.Round(float64(geometry.Height)/2)
    screenMidX:=math.Round(float64(wm.width)/2)
    screenMidY:=math.Round(float64(wm.height)/2)
    topLeftX := screenMidX-windowMidX
    topLeftY := screenMidY-windowMidY

    err = xproto.CreateWindowChecked(
        wm.conn,
        0,
        frameId,
        wm.root,
        int16(topLeftX),
        int16(topLeftY),
        geometry.Width,
        geometry.Height,
        BorderWidth,
        xproto.WindowClassInputOutput,
        xproto.WindowNone,
        xproto.CwBackPixel|xproto.CwBorderPixel|xproto.CwEventMask,
        []uint32{
            Col, // background (BG_COLOR)
            Col, // border color (BORDER_COLOR)
            xproto.EventMaskSubstructureRedirect |
            xproto.EventMaskSubstructureNotify | xproto.EventMaskEnterWindow | xproto.EventMaskLeaveWindow | xproto.EventMaskKeyPress | xproto.EventMaskButtonPress | xproto.EventMaskKeyRelease | xproto.EventMaskButtonRelease | xproto.EventMaskPointerMotion,
        },
    ).Check()

    if err!=nil{
        slog.Error("Couldn't create new window","error:", err.Error())
        return
    } 

    err = xproto.ChangeSaveSetChecked(
        wm.conn,
        xproto.SetModeInsert, // add to save set
        w,         // the client's window ID
    ).Check()

    if err!=nil{
        slog.Error("Couldn't save window to set","error:", err.Error())
        return
    }


    err = xproto.ReparentWindowChecked(
        wm.conn, 
        w,
        frameId, 
        0,0,
    ).Check()

    if err!=nil{
        slog.Error("Couldn't reparent window","error:", err.Error())
        return
    }

    err = xproto.MapWindowChecked(
        wm.conn,
        frameId,
    ).Check()

    if err!=nil{
        slog.Error("Couldn't map window","error:", err.Error())
        return
    }
    wm.currWorkspace.clients[w] = frameId
    wm.currWorkspace.frametoclient[frameId] = w
    wm.currWorkspace.windows[frameId] = &Window{
        X: int(topLeftX),
        Y: int(topLeftY),
        Width: int(geometry.Width),
        Height: int(geometry.Height),
        Fullscreen: false,
    }
    fmt.Println("Framed window"+strconv.Itoa(int(w))+"["+strconv.Itoa(int(frameId))+"]")
}

func (wm *WindowManager) OnConfigureRequest(event xproto.ConfigureRequestEvent){

    if frame, ok := wm.currWorkspace.clients[event.Window]; ok {
        changes := createChanges(event)

        xproto.ConfigureWindow(wm.conn, frame, event.ValueMask, changes)
        slog.Info("Resize", "frame", frame, "width", event.Width, "height", event.Height)
        return
    }


    changes:=createChanges(event)
    
    fmt.Println(event.ValueMask)
    fmt.Println(changes)

    err := xproto.ConfigureWindowChecked(
        wm.conn,
        event.Window,
        event.ValueMask,
        changes,
    ).Check()

    if err!=nil{
        slog.Error("couldn't configure window","error:", err.Error())
    }
}

func createChanges(event xproto.ConfigureRequestEvent) []uint32{
    changes := make([]uint32, 0, 7)

    if event.ValueMask&xproto.ConfigWindowX != 0 {
        changes = append(changes, uint32(event.X))
    }
    if event.ValueMask&xproto.ConfigWindowY != 0 {
        changes = append(changes, uint32(event.Y))
    }
    if event.ValueMask&xproto.ConfigWindowWidth != 0 {
        changes = append(changes, uint32(event.Width))
    }
    if event.ValueMask&xproto.ConfigWindowHeight != 0 {
        changes = append(changes, uint32(event.Height))
    }
    if event.ValueMask&xproto.ConfigWindowBorderWidth != 0 {
        changes = append(changes, uint32(event.BorderWidth))
    }
    if event.ValueMask&xproto.ConfigWindowSibling != 0 {
        changes = append(changes, uint32(event.Sibling))
    }
    if event.ValueMask&xproto.ConfigWindowStackMode != 0 {
        changes = append(changes, uint32(event.StackMode))
    }

    return changes
}

func (wm *WindowManager) Close(){
    if wm.conn != nil{
        wm.conn.Close()
    }
}


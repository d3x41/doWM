package wm

import (
    "fmt"
    "strconv"
    "os/exec"
    "log/slog"
    "math"
    "github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
    "github.com/BurntSushi/xgbutil"
    "github.com/BurntSushi/xgbutil/keybind"
)

var (
    XUtil *xgbutil.XUtil
)

type WindowManager struct{
    conn *xgb.Conn
    root xproto.Window
    clients map[xproto.Window]xproto.Window
    frametoclient map[xproto.Window]xproto.Window
    width, height int
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

    return &WindowManager{
        conn: X,
        root: root,
        clients: map[xproto.Window]xproto.Window{},
        frametoclient: map[xproto.Window]xproto.Window{},
        width: int(dimensions.Width),
        height: int(dimensions.Height),
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
        wm.Frame(window, true)
    }

    err = xproto.UngrabServerChecked(wm.conn).Check()

    if err!=nil{
        slog.Error("couldn't ungrab server", "error:", err.Error())
        return
    }

    cKeyCode := keybind.StrToKeycodes(XUtil, "c")[0]
    wKeyCode := keybind.StrToKeycodes(XUtil, "w")[0]
    f1KeyCode := keybind.StrToKeycodes(XUtil, "f1")[0]
    f2KeyCode := keybind.StrToKeycodes(XUtil, "f2")[0]
    f3KeyCode := keybind.StrToKeycodes(XUtil, "f3")[0]

    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1, cKeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1, wKeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1, f1KeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1, f2KeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
    err = xproto.GrabKeyChecked(wm.conn, true, wm.root, xproto.ModMask1, f3KeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()

    err = xproto.GrabButtonChecked(wm.conn, true, wm.root, 	uint16(xproto.EventMaskButtonPress | xproto.EventMaskButtonRelease | xproto.EventMaskPointerMotion), xproto.GrabModeAsync, xproto.GrabModeAsync, xproto.WindowNone, xproto.AtomNone, xproto.ButtonIndex1, xproto.ModMask1).Check()

    err = xproto.GrabButtonChecked(wm.conn, true, wm.root, 	uint16(xproto.EventMaskButtonPress | xproto.EventMaskButtonRelease | xproto.EventMaskPointerMotion), xproto.GrabModeAsync, xproto.GrabModeAsync, xproto.WindowNone, xproto.AtomNone, xproto.ButtonIndex3, xproto.ModMask1).Check()

    if err!=nil{
        slog.Error("couldn't grab window+c key", "error:", err.Error())
        return
    }

    var start xproto.ButtonPressEvent
    var attr *xproto.GetGeometryReply

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
                }
            case xproto.ButtonReleaseEvent:
                start.Child = 0
            case xproto.MotionNotifyEvent:
                ev := event.(xproto.MotionNotifyEvent)
                if start.Child != 0{
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
                        client:=wm.frametoclient[start.Child]
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
                if _, ok := wm.clients[ev.Window]; ok{
                    delete(wm.frametoclient, wm.clients[ev.Window])
                    delete(wm.clients, ev.Window)
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
                if ev.Detail==cKeyCode&&ev.State&xproto.ModMask1!=0{
                    SendWmDelete(wm.conn, wm.frametoclient[ev.Child])
                    fmt.Println(wm.frametoclient[ev.Child])
                    wm.UnFrame(wm.frametoclient[ev.Child])

                }
                if ev.Detail == cKeyCode && ev.State&(xproto.ModMask4|xproto.ModMaskShift) == (xproto.ModMask4|xproto.ModMaskShift) {
                    // Mod + Shift + C → force close
                    err := xproto.DestroyWindowChecked(wm.conn, wm.frametoclient[ev.Child]).Check()
                    if err != nil {
                        fmt.Println("Couldn't force destroy:", err)
                    }
                }
                if ev.Detail == wKeyCode&&ev.State&xproto.ModMask1!=0{
                    err := exec.Command("launcher_t2").Start()
                    if err != nil{
                        slog.Error("couldn't run rofi", "error:", err)
                    }
                }
                if ev.Detail == f1KeyCode&&ev.State&xproto.ModMask1!=0{
                    err := exec.Command("pactl", "set-sink-mute", "@DEFAULT_SINK@", "toggle").Start()
                    if err != nil{
                        slog.Error("couldn't toggle mute", "error:", err)
                    }   
                }
                if ev.Detail == f2KeyCode&&ev.State&xproto.ModMask1!=0{
                    err := exec.Command("pactl", "set-sink-volume", "@DEFAULT_SINK@", "-5%").Start()
                    if err != nil{
                        slog.Error("couldn't decrease volume", "error:", err)
                    }
                }
                if ev.Detail == f3KeyCode&&ev.State&xproto.ModMask1!=0{
                    err := exec.Command("pactl", "set-sink-volume", "@DEFAULT_SINK@", "+5%").Start()
                    if err != nil{
                        slog.Error("couldn't increase volume", "error:", err)
                    }
                } 
                break
            case xproto.ClientMessageEvent:
                ev := event.(xproto.ClientMessageEvent)
                atomName, _ := xproto.GetAtomName(wm.conn, xproto.Atom(ev.Type)).Reply()
                fmt.Println("Atom name is:", atomName.Name)
            default:
                fmt.Println("event: "+event.String())
                fmt.Println(event.Bytes())

        }
    }
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

func (wm *WindowManager) OnUnmapNotify(event xproto.UnmapNotifyEvent){
    if _, ok := wm.clients[event.Window]; !ok{
        slog.Info("couldn't unmap since window wasn't in clients")
        fmt.Println(event.Window)
        fmt.Println(wm.clients)
        return
    }

    if(event.Event == wm.root){
        slog.Info("Ignore UnmapNotify for reparented pre-existing window")
        fmt.Println(event.Window)
        return
    }

    wm.UnFrame(event.Window)
}

func (wm *WindowManager) UnFrame(w xproto.Window){
    frame := wm.clients[w]

    err := xproto.UnmapWindowChecked(
        wm.conn,
        frame,
    ).Check()

    if err!=nil{
        slog.Error("couldn't unmap frame", "error:", err.Error())
        return
    }

    err = xproto.ReparentWindowChecked(
        wm.conn, 
        w,
        wm.root,
        0, 0,
    ).Check()
    if err!=nil{
        slog.Error("couldn't remap window during unmapping", "error:", err.Error())
        return
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

    delete(wm.clients, w)
    delete(wm.frametoclient, frame)
    slog.Info("Unmapped", "frame", frame, "window", w)
}

func (wm *WindowManager) OnMapRequest(event xproto.MapRequestEvent){
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

    if _, exists := wm.clients[w]; exists {
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
    wm.clients[w] = frameId
    wm.frametoclient[frameId] = w
    fmt.Println("Framed window"+strconv.Itoa(int(w))+"["+strconv.Itoa(int(frameId))+"]")
}

func (wm *WindowManager) OnConfigureRequest(event xproto.ConfigureRequestEvent){

    if frame, ok := wm.clients[event.Window]; ok {
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


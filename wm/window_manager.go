package wm

import (	
    "github.com/mattn/go-shellwords"
	"encoding/binary"
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/keybind"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
)

var (
    XUtil *xgbutil.XUtil
)

var k = koanf.New(".")
type Config struct{
    Gap uint32 `koanf:"gaps"`
    BorderUnactive uint32 `koanf:"unactive-border-color"`
    BorderActive uint32 `koanf:"active-border-color"`
    ModKey string `koanf:"ModKey"`
    BorderWidth uint32 `koanf:"border-width"`
    Keybinds []Keybind `koanf:"keybinds"`
}


type Keybind struct{
    Keycode uint32
    Key string `koanf:"key"`
    Shift bool `koanf:"shift"`
    Exec string `koanf:"exec"`
    Role string `koanf:"role"`
}
type LayoutWindow struct{
    WidthPercentage, HeightPercentage, XPercentage, YPercentage float64
}

type Layout struct{
    Windows []LayoutWindow
}

type Window struct{
    id xproto.Window
    X,Y int
    Width, Height int
    Fullscreen bool
}

type Space struct{
    X,Y int
    Width, Height int
}

type Workspace struct{
    clients map[xproto.Window]xproto.Window
    frametoclient map[xproto.Window]xproto.Window
    windows map[xproto.Window]*Window
    tiling bool
}

type WindowManager struct{
    conn *xgb.Conn
    root xproto.Window
    width, height int
    workspaces []Workspace
    workspaceIndex int
    currWorkspace *Workspace
    atoms map[string]xproto.Atom
    tiling bool
    tilingspace Space
    layouts []Layout
    config Config
    mod uint16
}

func createLayouts() ([]Layout){
    return []Layout{
        {
            []LayoutWindow{ // one window layout
                {
                    XPercentage: 0, // left
                    YPercentage: 0, // top
                    WidthPercentage: 1, // full width
                    HeightPercentage: 1, // full height
                },
            },
        }, 
        {
            []LayoutWindow{ // two window layout
                {
                    XPercentage: 0, // left
                    YPercentage: 0, // top
                    WidthPercentage: 0.5, // half width
                    HeightPercentage: 1, // full height
                },
                {
                    XPercentage: 0.5, // half from left
                    YPercentage: 0, // top
                    WidthPercentage: 0.5, // half width
                    HeightPercentage: 1, // full height
                },
            },
        },
        {
            []LayoutWindow{ // three window layout
                {
                    XPercentage: 0, //left
                    YPercentage: 0, //top
                    WidthPercentage: 1.0/3, // third of width
                    HeightPercentage: 1, // full height
                },
                {
                    XPercentage: 1.0/3, // third from left
                    YPercentage: 0, // top
                    WidthPercentage: 1.0/3, //third of width
                    HeightPercentage: 1, // full height
                },
                {
                    XPercentage: 2.0/3, // 2 thirds from left
                    YPercentage: 0, // top
                    WidthPercentage: 1.0/3, // third of width
                    HeightPercentage: 1, // full height
                },
            },
        },
        {
            []LayoutWindow{ // 4 window layout
                {
                    XPercentage: 0, // left
                    YPercentage: 0, // top
                    WidthPercentage: 0.5, // half width
                    HeightPercentage: 0.5, // half height
                },
                {
                    XPercentage: 0.5, // half from left
                    YPercentage: 0, // top
                    WidthPercentage: 0.5, // half width
                    HeightPercentage: 0.5, // half height
                },
                {
                    XPercentage: 0, //left
                    YPercentage: 0.5, // half from top
                    WidthPercentage: 0.5, // half width
                    HeightPercentage: 0.5, // half height
                },
                {
                    XPercentage: 0.5, // half from left
                    YPercentage: 0.5, // half from top
                    WidthPercentage: 0.5, // half width
                    HeightPercentage: 0.5,// half height
                },
            },
        },
    }
}

func createConfig(f koanf.Provider) Config{ 
    // Set defaults manually
    cfg := Config{
        Gap:            6,
        BorderWidth:    3,
        ModKey:         "Mod1",
        BorderUnactive: 0x8bd5ca,
        BorderActive:   0xa6da95,
        Keybinds: []Keybind{},
    }

    // Load the config file
    if err := k.Load(f, yaml.Parser()); err == nil {
        // Unmarshal — existing keys override the defaults
        k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{Tag: "koanf", FlatPaths: false})

    }else{
        slog.Warn("couldn't load config, using defaults")
    }

    return cfg
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
            tiling: false,
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
        tiling: false,
        layouts: createLayouts(),
    }, nil
}
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func (wm *WindowManager) createKeybind(kb *Keybind) Keybind{
    code := keybind.StrToKeycodes(XUtil, kb.Key)
    if len(code)<1{
        return Keybind{
            Keycode: 0,
            Key: "",
            Shift: false,
            Exec: "",
        }
    }
    KeyCode := code[0]
    kb.Keycode=uint32(KeyCode)
    Mask := wm.mod
    if kb.Shift{
        Mask = Mask | xproto.ModMaskShift
    }
    err := xproto.GrabKeyChecked(wm.conn, true, wm.root, Mask, KeyCode , xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
    if err != nil{
        slog.Error("couldn't create keybind", "error:", err)
    }

    return *kb
}

func (wm *WindowManager) Run(){
    fmt.Println("window manager up and running")

    user, err := user.Current()
    if err == nil{
        scriptPath := filepath.Join(user.HomeDir, ".config", "doWM", "autostart.sh")

        if fileExists(scriptPath){
            fmt.Println("autostart exists..., running")
            exec.Command("/home/sam/.config/doWM/autostart.sh").Start()
        } 
    }
    err = xproto.ChangeWindowAttributesChecked(
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

    home, _ := os.UserHomeDir()
    f := file.Provider(filepath.Join(home, ".config", "doWM", "doWM.yml"))
    cfg:=createConfig(f)
    wm.config = cfg
    //TODO: make auto-reload

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

    var mMask uint16
    switch(wm.config.ModKey){
        case "Mod1":
            mMask=xproto.ModMask1
        case "Mod2":
            mMask=xproto.ModMask2
        case "Mod3":
            mMask=xproto.ModMask3
        case "Mod4":
            mMask=xproto.ModMask4
        case "Mod5":
            mMask=xproto.ModMask5 
    }

    wm.mod = mMask
    
    for i, kb := range wm.config.Keybinds{
        wm.config.Keybinds[i] = wm.createKeybind(&kb)
    }
    /*
    wm.createKeybind(&Keybind{ Key: "w", Shift: false , Keycode: 0})*/
    wm.config.Keybinds = append(wm.config.Keybinds, []Keybind{
    wm.createKeybind(&Keybind{ Key: "c", Shift: false , Keycode: 0}),
    wm.createKeybind(&Keybind{ Key: "c", Shift: true , Keycode: 0}),
    wm.createKeybind(&Keybind{ Key: "v", Shift: false , Keycode: 0}),
    wm.createKeybind(&Keybind{ Key: "f", Shift: false, Keycode: 0}),
    wm.createKeybind(&Keybind{ Key: "0", Shift: false , Keycode: 0}),
    wm.createKeybind(&Keybind{ Key: "1", Shift: false , Keycode: 0}),
    wm.createKeybind(&Keybind{ Key: "2", Shift: false , Keycode: 0}),
    wm.createKeybind(&Keybind{ Key: "3", Shift: false , Keycode: 0}),
    wm.createKeybind(&Keybind{ Key: "4", Shift: false , Keycode: 0}),
    wm.createKeybind(&Keybind{ Key: "5", Shift: false , Keycode: 0}),
    wm.createKeybind(&Keybind{ Key: "6", Shift: false , Keycode: 0}),
    wm.createKeybind(&Keybind{ Key: "7", Shift: false , Keycode: 0}),
    wm.createKeybind(&Keybind{ Key: "8", Shift: false , Keycode: 0}),
    wm.createKeybind(&Keybind{ Key: "9", Shift: false , Keycode: 0}),
    wm.createKeybind(&Keybind{ Key: "0", Shift: true , Keycode: 0}),
    wm.createKeybind(&Keybind{ Key: "1", Shift: true , Keycode: 0}),
    wm.createKeybind(&Keybind{ Key: "2", Shift: true , Keycode: 0}),
    wm.createKeybind(&Keybind{ Key: "3", Shift: true , Keycode: 0}),
    wm.createKeybind(&Keybind{ Key: "4", Shift: true , Keycode: 0}),
    wm.createKeybind(&Keybind{ Key: "5", Shift: true , Keycode: 0}),
    wm.createKeybind(&Keybind{ Key: "6", Shift: true , Keycode: 0}),
    wm.createKeybind(&Keybind{ Key: "7", Shift: true , Keycode: 0}),
    wm.createKeybind(&Keybind{ Key: "8", Shift: true , Keycode: 0}),
    wm.createKeybind(&Keybind{ Key: "9", Shift: true , Keycode: 0}),
    }...)

    fmt.Println(wm.config.Keybinds)

    err = xproto.GrabButtonChecked(wm.conn, true, wm.root, 	uint16(xproto.EventMaskButtonPress | xproto.EventMaskButtonRelease | xproto.EventMaskPointerMotion), xproto.GrabModeAsync, xproto.GrabModeAsync, xproto.WindowNone, xproto.AtomNone, xproto.ButtonIndex1, mMask).Check()

    err = xproto.GrabButtonChecked(wm.conn, true, wm.root, 	uint16(xproto.EventMaskButtonPress | xproto.EventMaskButtonRelease | xproto.EventMaskPointerMotion), xproto.GrabModeAsync, xproto.GrabModeAsync, xproto.WindowNone, xproto.AtomNone, xproto.ButtonIndex3, mMask).Check()

    if err!=nil{
        slog.Error("couldn't grab window+c key", "error:", err.Error())
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
        "_NET_WM_WINDOW_TYPE",
        "_NET_WM_WINDOW_TYPE_DOCK",
        "_NET_WM_STRUT_PARTIAL",
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
                if start.Child != 0&&ev.State&mMask!=0{
                    if wm.tiling || (wm.currWorkspace.windows[start.Child]!=nil&&wm.currWorkspace.windows[start.Child].Fullscreen){
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
                    wm.UnFrame(wm.currWorkspace.clients[ev.Window], true)
                    wm.fitToLayout()
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
                if ev.State&mMask!=0{
                    for _, kb := range wm.config.Keybinds{
                        if(ev.Detail == xproto.Keycode(kb.Keycode)&&(ev.State&(mMask|xproto.ModMaskShift) == (mMask|xproto.ModMaskShift)==kb.Shift)){

                            if kb.Exec!=""{
                                fmt.Println("executing:", kb.Exec)
                                runCommand(kb.Exec)
                            }
                            switch(kb.Role){
                                case "quit":
                                        SendWmDelete(wm.conn, wm.currWorkspace.frametoclient[ev.Child])
                                        fmt.Println(wm.currWorkspace.frametoclient[ev.Child])
                                        break
                                case "force-quit":
                                    // Mod + Shift + C → force close
                                    err := xproto.DestroyWindowChecked(wm.conn, wm.currWorkspace.frametoclient[ev.Child]).Check()
                                    if err != nil {
                                        fmt.Println("Couldn't force destroy:", err)
                                    }
                                    break
                                case "toggle-tiling":
                                    wm.toggleTiling()
                                    break
                                case "toggle-fullscreen":
                                    wm.toggleFullScreen(ev.Child)
                            }
                            switch(kb.Key){
                                case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
                                    w := ev.Child
                                    var client xproto.Window
                                    var window Window
                                    if kb.Shift{
                                        client = wm.currWorkspace.frametoclient[w]
                                        window = *wm.currWorkspace.windows[w]
                                        fmt.Println("moving window")
                                        xproto.ConfigureWindow(
                                            wm.conn,
                                            w,
                                            xproto.ConfigWindowStackMode,
                                            []uint32{xproto.StackModeAbove},
                                        )
                                        delete(wm.currWorkspace.clients, wm.currWorkspace.frametoclient[w])
                                        delete(wm.currWorkspace.windows, w)
                                        delete(wm.currWorkspace.frametoclient, w)
                                    }                        
                                    switch kb.Key{
                                    case "1":
                                        wm.switchWorkspace(0)
                                    case "2":
                                        wm.switchWorkspace(1)
                                    case "3":
                                        wm.switchWorkspace(2)
                                    case "4":
                                        wm.switchWorkspace(3)
                                    case "5":
                                        wm.switchWorkspace(4)
                                    case "6":
                                        wm.switchWorkspace(5)
                                    case "7":
                                        wm.switchWorkspace(6)
                                    case "8":
                                        wm.switchWorkspace(7)
                                    case "9":
                                        wm.switchWorkspace(8)
                                    case "0":
                                        wm.switchWorkspace(9)
                                    }
                                    if kb.Shift{
                                        wm.currWorkspace.frametoclient[w]=client
                                        wm.currWorkspace.windows[w]=&window
                                        wm.currWorkspace.clients[client]=w 
                                    }
                                    wm.fitToLayout()
                                    
                                    break
                                }
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
func runCommand(cmdStr string) {
	parser := shellwords.NewParser()
	args, err := parser.Parse(cmdStr)
	if err != nil {
        slog.Error("parse error:", "error:", err)
		return
	}
    if len(args)<2{
        cmd := exec.Command(args[0])
        cmd.Run()
        return
    }
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Run()
}
func (wm *WindowManager) getBar(vals []byte) (int, int, int, int){
    var maxLeft, maxRight, maxTop, maxBottom int
    left := int(binary.LittleEndian.Uint32(vals[0:4]))
    right := int(binary.LittleEndian.Uint32(vals[4:8]))
    top := int(binary.LittleEndian.Uint32(vals[8:12]))
    bottom := int(binary.LittleEndian.Uint32(vals[12:16]))

    if left > maxLeft {
        maxLeft = left
    }
    if right > maxRight {
        maxRight = right
    }
    if top > maxTop {
        maxTop = top
    }
    if bottom > maxBottom {
        maxBottom = bottom
    }
    return maxLeft, maxRight, maxTop, maxBottom
}

func (wm *WindowManager) createTilingSpace(){
    windows,_ := xproto.QueryTree(wm.conn, wm.root).Reply()
    X := 0
    Y := 0
    width := wm.width
    height := wm.height

    for _, window := range windows.Children{
        attributes, _ := xproto.GetWindowAttributes(wm.conn, window).Reply()
        if attributes.MapState == xproto.MapStateViewable{
            atom := wm.atoms["_NET_WM_STRUT_PARTIAL"]
            prop, err := xproto.GetProperty(wm.conn, false, window, atom, xproto.AtomCardinal, 0, 12).Reply()

            if err != nil || prop == nil || prop.ValueLen < 4{
                continue
            }	

            vals := prop.Value
            if len(vals) < 16 {
                continue // need at least 4 uint32s
            }
            left, right, top, bottom := wm.getBar(vals)

            X = left
            Y = top
            width = wm.width - left - right
            height = wm.height - top - bottom

            // TODO: support multiple bars
            break
        }
    }

    fmt.Println("tiling container:", "X:", X, "Y:", Y, "Width:", width, "Height:", height)
    wm.tilingspace = Space{
        X: X,
        Y: Y,
        Width: width-6,
        Height: height-6,
    }
}

func (wm *WindowManager) fitToLayout(){
    if !wm.tiling{
        return
    }
    windowNum := len(wm.currWorkspace.frametoclient)
    if windowNum >4||windowNum<1{
        return
    }
    layout := wm.layouts[windowNum-1]
    i:=0
    for window, WindowData := range wm.currWorkspace.windows{
        if WindowData.Fullscreen{
            continue
        }
        layoutWindow := layout.Windows[i]
        X := wm.tilingspace.X+int((float64(wm.tilingspace.Width)*layoutWindow.XPercentage))+int(wm.config.Gap)
        Y := wm.tilingspace.Y+int((float64(wm.tilingspace.Height)*layoutWindow.YPercentage))+int(wm.config.Gap)
        Width := (float64(wm.tilingspace.Width)*layoutWindow.WidthPercentage)-float64(wm.config.Gap*2)
        Height := (float64(wm.tilingspace.Height)*layoutWindow.HeightPercentage)-float64(wm.config.Gap*2)
        fmt.Println("window:", window, "X:", X, "Y:", Y, "Width:", Width, "Height:", Height)
        wm.configureWindow(window, X, Y, int(Width), int(Height))
        i++
    }
}

func (wm *WindowManager) configureWindow(Frame xproto.Window, X, Y, Width, Height int){
        err := xproto.ConfigureWindowChecked(wm.conn, Frame, xproto.ConfigWindowX | xproto.ConfigWindowY | xproto.ConfigWindowWidth|xproto.ConfigWindowHeight, []uint32{
            uint32(X), uint32(Y), uint32(Width), uint32(Height),
        }).Check()
        if err != nil{
            slog.Error("couldn't configure window!", "error:", err)
            return
        }
        tree, _ := xproto.QueryTree(wm.conn, Frame).Reply()
        if len(tree.Children)>0{
            child := tree.Children[0]
            err = xproto.ConfigureWindowChecked(wm.conn, child, xproto.ConfigWindowX | xproto.ConfigWindowY | xproto.ConfigWindowWidth|xproto.ConfigWindowHeight, []uint32{
                0, 0, uint32(Width), uint32(Height),
            }).Check()
            if err != nil{
                slog.Error("couldn't configure window!", "error:", err)
                return
            }
        }
}

func (wm *WindowManager) toggleTiling(){
    if !wm.tiling{
        wm.tiling=true
        wm.enableTiling()
    }else{
        wm.tiling=false
        wm.disableTiling()
    }
}

func (wm *WindowManager) disableTiling(){
        wm.currWorkspace.tiling = false
        fmt.Println("DISABLED TILING")
        fmt.Println(len(wm.currWorkspace.windows))
        for windowId, window := range wm.currWorkspace.windows{
            if window.Fullscreen{
                fmt.Println("from: disable tiling, toggling fullscreen on window", windowId)
                wm.toggleFullScreen(windowId)
            }
            wm.configureWindow(windowId, window.X, window.Y, window.Width, window.Height)
        }
}

func (wm *WindowManager) enableTiling(){
        wm.currWorkspace.tiling = true
        for windowId, window := range wm.currWorkspace.windows{
            if window.Fullscreen{
                fmt.Println("from: enable tiling, toggling fullscreen on window", windowId)
                wm.toggleFullScreen(windowId)
            }
            attr, _ := xproto.GetGeometry(wm.conn, xproto.Drawable(windowId)).Reply()
            wm.currWorkspace.windows[windowId] = &Window{
                id: window.id,
                X:int(attr.X),
                Y:int(attr.Y),
                Width: int(attr.Width),
                Height: int(attr.Height),
                Fullscreen: false,
            }
        }
        fmt.Println("tiling")
        wm.createTilingSpace()
        wm.fitToLayout()
}

func (wm *WindowManager) clearTileContainer(){
    for window, windowVal := range wm.currWorkspace.windows{
        err := xproto.ReparentWindowChecked(wm.conn, window, wm.root,int16(windowVal.X), int16(windowVal.Y)).Check()
        if err != nil{
            slog.Error("couldn't reparent window", "error:", err)
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
                []uint32{uint32(win.X), uint32(win.Y), uint32(win.Width), uint32(win.Height), wm.config.BorderWidth},
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
            wm.fitToLayout()
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
        xproto.UnmapWindowChecked(wm.conn, frame)
    }

    wm.currWorkspace = &wm.workspaces[workspace]
    wm.workspaceIndex = workspace

    for frame := range wm.currWorkspace.frametoclient{
        xproto.MapWindowChecked(wm.conn, frame)
    }

    wm.conn.Sync()

    if wm.tiling && !wm.currWorkspace.tiling{
        wm.enableTiling()
    }else if !wm.tiling && wm.currWorkspace.tiling{
        wm.disableTiling()
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
    Col := wm.config.BorderUnactive

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
    Col := wm.config.BorderActive
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
            delete(wm.currWorkspace.windows, frame)
            wm.workspaces[wm.workspaceIndex].frametoclient[frame]=client
            wm.workspaces[wm.workspaceIndex].clients[client]=frame 
            fmt.Println("frame")
            fmt.Println(frame)
            fmt.Println("index")
            fmt.Println(index)
            wm.currWorkspace = &wm.workspaces[wm.workspaceIndex]
            wm.UnFrame(wm.currWorkspace.frametoclient[frame], true)
            wm.fitToLayout()
            return
        }
    }

    if(event.Event == wm.root){
        slog.Info("Ignore UnmapNotify for reparented pre-existing window")
        fmt.Println(event.Window)
        return
    }

    wm.UnFrame(event.Window, false)
    wm.fitToLayout()
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
    netWmPanel, err := xproto.InternAtom(conn, false, uint16(len("_NET_WM_WINDOW_TYPE_PANEL")), "_NET_WM_WINDOW_TYPE_PANEL").Reply()
    if err != nil {
        slog.Error("Error getting _NET_WM_WINDOW_TYPE_PANEL atom", "error", err)
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

    if windowType == netWmSplash.Atom || windowType == netWmDialog.Atom || windowType == netWmNotification.Atom || windowType == netWmDock.Atom||windowType==netWmPanel.Atom {
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
    if wm.tiling{
        wm.fitToLayout()
    }
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
    BorderWidth := wm.config.BorderWidth
    Col := wm.config.BorderUnactive

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
        uint16(BorderWidth),
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


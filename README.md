<div align="center">
      <h1>doWM</h1>
     </div>
<p align="center"> <a href="https://github.com/BobdaProgrammer/doWM" target="_blank"><img alt="" src="https://img.shields.io/badge/Github-302D41?style=for-the-badge&logo=github" style="vertical-align:center" /></a>
</p>
<p align="center">
    <a href="https://github.com/BobdaProgrammer/doWM/pulse" target="_blank"><img src="https://img.shields.io/github/last-commit/BobdaProgrammer/doWM?style=for-the-badge&logo=github&color=7dc4e4&logoColor=D9E0EE&labelColor=302D41"></a>
    <a href="https://github.com/BobdaProgrammer/doWM/stargazers" target="_blank"><img src="https://img.shields.io/github/stars/BobdaProgrammer/doWM?style=for-the-badge&logo=apachespark&color=eed49f&logoColor=D9E0EE&labelColor=302D41"></a>
</p><p align="center">
      <a href="https://visitorbadge.io/status?path=https%3A%2F%2Fgithub.com%2FBobdaProgrammer%2FdoWM"><img src="https://api.visitorbadge.io/api/visitors?path=https%3A%2F%2Fgithub.com%2FBobdaProgrammer%2FdoWM&label=visitors&labelColor=%23ff8a65&countColor=%23111133" /></a>
      <a href="https://github.com/BobdaProgrammer/doWM/issues" target="_blank">
      <img alt="Issues" src="https://img.shields.io/github/issues/BobdaProgrammer/doWM?style=for-the-badge&logo=bilibili&color=F5E0DC&logoColor=D9E0EE&labelColor=302D41" />
    </a>  
       <a href="https://github.com/BobdaProgrammer/doWM/blob/main/LICENSE" target="_blank">
      <img alt="License" src="https://img.shields.io/github/license/BobdaProgrammer/doWM?style=for-the-badge&logo=starship&color=ee999f&logoColor=D9E0EE&labelColor=302D41" />
    </a>  
    <a href="https://github.com/BobdaProgrammer/doWM" target="_blank">
      <img alt="Repo Size" src="https://img.shields.io/github/repo-size/BobdaProgrammer/doWM?color=%23DDB6F2&label=SIZE&logo=codesandbox&style=for-the-badge&logoColor=D9E0EE&labelColor=302D41" />
    </a>
</p>

## Description
doWM is a mainly floating but also tiling window manager for X11 completely written in golang.

## Installation
Currently the best way is to build from source:

You will want to have go installed

```bash
git clone https://github.com/BobdaProgrammer/doWM
cd doWM
make build
make install
```

then to see a normal config look at `exampleConfig` folder, you can copy this to ~/.config/doWM and then write your own configuration

## Configuration
doWM is configured with `~/.config/doWM/doWM.yml` and `~/.confiig/doWM/autostart.sh`
simply put any autostart commands in autostart.sh, and then remember to chmod +x it.
the main config file is very simple and is described clearly in the comments on /exampleConfig/doWM.yml

Colors are to be written in hex format starting with 0x for example white: 0xffffff (could be 0xFFFFFF it is case insensitive)

You have a few general options:
- gaps (pixel gaps in tiling)
- mod-key (which key should be used for all wm commands)
- border-width (border width of windows)
- unactive-border-color (the color for the border of unactive windows
- active-border-color (the color for the border of an active window)

there are some default keybinds like modkey+(0-9) to switch workspaces and with a shift to move a window between workspaces

then there are keybinds, each keybind either executes a command or plays a role in the wm. Here are all the roles:
- quit (close window)
- force-quit (force close window)
- toggle-tiling (toggle tiling mode)
- toggle-fullscreen (toggle fullscreen on window)
- swap-window-left (shift window left in tiling mode)
- swap-window-right (shift window right in tiling mode)
- reload-config (reload doWM.yml)

each keybind also has a key and a shift option, key is the character of the key (can also be things like "F1") and shift is a bool for if shift should be pressed or not to register.

Below is the example config:
```yml
# gaps for tiling windows
gaps: 10

# the mod key used for all window manager actions
# Mod1 = alt
# Mod4 = windows/super key
# those are the usual although all 1-5 are supported
mod-key: "Mod1"

border-width: 5

# border color for unfocused windows
unactive-border-color: 0xBBFFDC

# border color for focused windows
active-border-color: 0xEEFFBB

# keybindings
# follow this pattern
# a key (can't be multiple) in lowercase
# wether the kebybind is with shift
# either a command to exec or a role in the window manager
# roles are:
# - quit
# - force-quit
# - toggle-tiling
# - toggle-fullscreen
# - swap-window-left
# - swap-window-right
# - reload-config
keybinds:
  - key: "w"
    shift: false
    exec: "rofi -show drun"
  - key: "t"
    shift: false
    exec: "kitty"
  - key: "e"
    shift: false
    exec: "thunar"
  - key: "f1"
    shift: false
    exec: "pactl set-sink-mute @DEFAULT_SINK@ toggle"
  - key: "f2"
    shift: false
    exec: "pactl set-sink-volume @DEFAULT_SINK@ -5%"
  - key: "f3"
    shift: false
    exec: "pactl set-sink-volume @DEFAULT_SINK@ +5%"
  - key: "c"
    shift: false
    role: "quit"
  - key: "c"
    shift: true
    role: "force-quit"
  - key: "f"
    shift: false
    role: "toggle-fullscreen"
  - key: "v"
    shift: false
    role: "toggle-tiling"
  - key: "left"
    shift: false
    role: "swap-window-left"
  - key: "right"
    shift: false
    role: "swap-window-right"
  - key: "r"
    shift: true
    role: "reload-config"
```

## screenshots
<div align="center">
<img src="https://github.com/BobdaProgrammer/doWM/blob/main/images/floating.png?raw=true" width="500px">
<img src="https://github.com/BobdaProgrammer/doWM/blob/main/images/tiling.png?raw=true" width="500px">
  
<img src="https://github.com/BobdaProgrammer/doWM/blob/main/images/floatingTerminals.png?raw=true" width="500px">
<img src="https://github.com/BobdaProgrammer/doWM/blob/main/images/tilingTerminals.png?raw=true" width="500px">

<img src="https://github.com/BobdaProgrammer/doWM/blob/main/images/rofi.png?raw=true" width="500px">
<img src="https://github.com/BobdaProgrammer/doWM/blob/main/images/musicWindow.png?raw=true" width="500px">
</div>  

-------------

## progress
- [x] move/resize
- [x] workspaces
- [x] move window between workspaces
- [x] focus on hover
- [x] configuration
- [x] keybinds
- [x] floating
- [x] tiling
- [x] bar support
- [x] fullscreen
- [x] startup commands
- [x] picom support

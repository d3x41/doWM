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
doWM is a beautiful floating and tiling window manager for X11 completely written in golang.

## Installation
Currently the best way is to build from source:

You will want to have go installed

```bash
git clone https://github.com/BobdaProgrammer/doWM
cd doWM
go build -o ./doWM
make install
```

then to see a normal config look at `exampleConfig` folder, you can copy this to ~/.config/doWM and then write your own configuration  

-------------

> [!WARNING]
> make sure to make the autostart.sh executable and to use a config, otherwise you could be left in the black with no way to escape!

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
- focus-window-left (focus the window to the left in tiling mode)
- focus-window-right (focus the window to the right in tiling mode)
- reload-config (reload doWM.yml)

each keybind also has a key and a shift option, key is the character of the key (can also be things like "F1") and shift is a bool for if shift should be pressed or not to register.

for example: 
```yml
  # When mod + t is pressed then open kitty
  - key: "t"
    shift: false
    exec: "kitty"
  # When mod + shift + right arrow is pressed then switch the focused window to the right
  - key: "right"
    shift: true
    role: "swap-window-right"
```

For an example config, look at [/exampleConfig](https://github.com/BobdaProgrammer/doWM/tree/main/exampleConfig)

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

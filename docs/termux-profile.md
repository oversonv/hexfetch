# Termux profile (Fastfetch-style icons)

A personal profile for Termux with a Fastfetch-like look. Save it as
`$HOME/hex/set/hexfetch.yaml`; it is layered on top of the built-in defaults.
Python is shown through the generic `command` module, so no built-in Python
module is required.

```sh
mkdir -p "$HOME/hex/set"
$EDITOR "$HOME/hex/set/hexfetch.yaml"   # paste the profile below
```

```yaml
hexfetch:
  logo: auto
  display:
    separator: " » "
    separator_char: "─"
    separator_length: 18
    key:
      type: both
    color:
      keys: "38;2;255;255;255"
      title: "38;2;255;255;255"
      output: "38;2;255;255;255"
      separator: "38;2;190;190;190"
  modules:
    items:
      - title
      - separator
      - os
      - packages
      - shell
      - command
      - terminal
    title:
      text: "You again?"
    os:
      key: "OS"
      key_icon: "\uf313"
    packages:
      key: "Packages"
      key_icon: "\uf487"
    shell:
      key: "Shell"
      key_icon: "\uf489"
    command:
      key: "Python"
      key_icon: "\ue73c"
      text: "python -V 2>&1 | cut -d' ' -f2"
    terminal:
      key: "Terminal"
      key_icon: "\uf489"
```

`hexfetch --gen-config` writes the built-in defaults instead and never
overwrites an existing file.

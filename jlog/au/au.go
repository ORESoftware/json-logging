package au

import (
  "os"
  au "github.com/logrusorgru/aurora/v4"
)

var colors = os.Getenv("vibe_with_color")
var hasColor = colors != "no"

var Col = au.New(au.WithColors(hasColor))

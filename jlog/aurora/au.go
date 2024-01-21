package au

import (
  "os"
  aurora "github.com/logrusorgru/aurora/v4"
)

var colors = os.Getenv("vibe_with_color")
var hasColor = colors != "no"

var Col = aurora.New(aurora.WithColors(hasColor))

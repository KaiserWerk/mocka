package assets

import _ "embed"

//go:embed files/consoleTemplate.txt
var ConsoleTemplate string

//go:embed files/webappTemplate.txt
var WebAppTemplate string

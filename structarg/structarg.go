package structarg

import (
    "os"
    "log"
    "bytes"
    "bufio"
    "fmt"
    "strings"
    "reflect"
    "strconv"
    "github.com/swordqiu/structarg.go/gotypes"
)

type Argument interface {
    NeedData() bool
    Token() string
    ShortToken() string
    MetaVar() string
    IsOptional() bool
    IsPositional() bool
    IsMulti() bool
    IsSubcommand() bool
    HelpString(indent string) string
    String() string
    SetValue(val string) error
    DoAction() error
    Validate() error
}

type SingleArgument struct {
    token string
    shortToken string
    metavar string
    optional bool
    positional bool
    help string
    choices []string
    useDefault bool
    defValue reflect.Value
    value reflect.Value
    isSet bool
    parser *ArgumentParser
}

type MultiArgument struct {
    SingleArgument
    minCount int64
    maxCount int64
}

type SubcommandArgumentData struct {
    parser *ArgumentParser
    callback reflect.Value
}

type SubcommandArgument struct {
    SingleArgument
    subcommands map[string]SubcommandArgumentData
}

type ArgumentParser struct {
    target interface{}
    prog string
    description string
    epilog string
    optArgs []Argument
    posArgs []Argument
}

func NewArgumentParser(target interface{}, prog, desc, epilog string) (*ArgumentParser, error) {
    parser := ArgumentParser{prog: prog, description: desc,
                            epilog: epilog, target: target}
    target_type := reflect.TypeOf(target).Elem()
    target_value := reflect.ValueOf(target).Elem()
    e := parser.addStructArgument(target_type, target_value)
    if e != nil {
        return nil, e
    }
    return &parser, nil
}

const (
    /*
    help text of the argument
    the argument is optional.
    */
    TAG_HELP = "help"
    /*
    command-line token for the optional argument, e.g. token:"url"
    the command-line argument will be "--url http://127.0.0.1:3306"
    the tag is optional.
    if the tag is missing, the variable name will be used as token.
    If the variable name is CamelCase, the token will be transformed
    into kebab-case, e.g. if the variable is "AuthURL", the token will
    be "--auth-url"
    */
    TAG_TOKEN = "token"
    /*
    short form of command-line token, e.g. short-token:"u"
    the command-line argument will be "-u http://127.0.0.1:3306"
    the tag is optional
    */
    TAG_SHORT_TOKEN = "short-token"
    /*
    Metavar of the argument
    the tag is optional
    */
    TAG_METAVAR = "metavar"
    /*
    The default value of the argument.
    the tag is optional
    */
    TAG_DEFAULT = "default"
    /*
    The possible values of an arguments. All choices are are concatenatd by "|".
    e.g. `choices:"1|2|3"`
    the tag is optional
    */
    TAG_CHOICES = "choices"
    /*
    A boolean value explicitly declare whether the argument is optional,
    the tag is optional
    */
    TAG_OPTIONAL = "optional"
    /*
    A boolean value explicitly decalre whther the argument is an subcommand
    A subcommand argument must be the last positional argument.
    the tag is optional, the default value is false
    */
    TAG_SUBCOMMAND = "subcommand"
    /*
    The attribute defines the possible number of argument. Possible values
    ar:
        * positive integers, e.g. "1", "2"
        * "*" any number of arguments
        * "+" at lease one argument
        * "?" at most one argument
    the tag is optional, the default value is "1"
    */
    TAG_NARGS = "nargs"
)

func (this *ArgumentParser) addStructArgument(tp reflect.Type, val reflect.Value) error {
    for i := 0; i < tp.NumField(); i ++ {
        f := tp.Field(i)
        v := val.Field(i)
        if f.Type.Kind() == reflect.Struct {
            return this.addStructArgument(f.Type, v)
        }else {
            e := this.addArgument(f, v)
            if e != nil {
                return e
            }
        }
    }
    return nil
}

func (this *ArgumentParser) addArgument(f reflect.StructField, v reflect.Value) error {
    help := f.Tag.Get(TAG_HELP)
    token := f.Tag.Get(TAG_TOKEN)
    if len(token) == 0 {
        token = f.Name
    }
    shorttoken := f.Tag.Get(TAG_SHORT_TOKEN)
    metavar := f.Tag.Get(TAG_METAVAR)
    defval := f.Tag.Get(TAG_DEFAULT)
    if len(defval) > 0 {
        for _, dv := range strings.Split(defval, "|") {
            if dv[0] == '$' {
                dv = os.Getenv(strings.TrimLeft(dv, "$"))
            }
            defval = dv
            if len(defval) > 0 {
                break
            }
        }
    }
    use_default := true
    if len(defval) == 0 {
        use_default = false
    }
    choices_str := f.Tag.Get(TAG_CHOICES)
    choices := make([]string, 0)
    if len(choices_str) > 0 {
        for _, s := range strings.Split(choices_str, "|") {
            if len(s) > 0 {
                choices = append(choices, s)
            }
        }
    }
    var positional, optional bool
    if f.Name == strings.ToUpper(f.Name) {
        positional = true
        optional = false
    }else {
        positional = false
        optional = true
    }
    opt_val := f.Tag.Get(TAG_OPTIONAL)
    if len(opt_val) > 0 {
        if opt_val == "true" {
            optional = true
        }else if opt_val == "false" {
            optional = false
        }
    }
    if positional && ! optional && use_default {
        return fmt.Errorf("A positional non-optional argument should not set default value")
    }
    subcommand, e := strconv.ParseBool(f.Tag.Get(TAG_SUBCOMMAND))
    if e != nil {
        subcommand = false
    }
    var defval_t reflect.Value
    if use_default {
        defval_t, e = gotypes.ParseValue(defval, f.Type)
        if e != nil {
            return e
        }
    }
    if subcommand {
        positional = true
        optional = false
    }
    var arg Argument = nil
    sarg := SingleArgument{token: token, shortToken: shorttoken,
                    optional: optional, positional: positional,
                    metavar: metavar, help: help,
                    choices: choices,
                    useDefault: use_default,
                    defValue: defval_t,
                    value: v, parser: this}
    if subcommand {
        arg = &SubcommandArgument{SingleArgument: sarg,
                        subcommands: make(map[string]SubcommandArgumentData)}
    }else if f.Type.Kind() == reflect.Array {
        var min, max int64
        var e error
        nargs := f.Tag.Get(TAG_NARGS)
        if nargs == "*" {
            min = 0
            max = -1
        }else if nargs == "?" {
            min = 0
            max = 1
        }else if nargs == "+" {
            min = 1
            max = -1
        }else {
            min, e = strconv.ParseInt(nargs, 10, 64)
            if e == nil {
                max = min
            }else {
                return fmt.Errorf("Unknown nargs pattern %s", nargs)
            }
        }
        arg = &MultiArgument{SingleArgument: sarg,
                    minCount: min, maxCount: max}
    }else {
        arg = &sarg
    }
    return this.AddArgument(arg)
}

func (this *ArgumentParser) AddArgument(arg Argument) error {
    if arg.IsPositional() {
        if len(this.posArgs) > 0 {
            last_arg := this.posArgs[len(this.posArgs)-1]
            switch {
                case last_arg.IsMulti():
                    return fmt.Errorf("Cannot append positional argument after an array positional argument")
                case last_arg.IsSubcommand():
                    return fmt.Errorf("Cannot append positional argument after a subcommand argument")
                case last_arg.IsOptional() && !arg.IsOptional():
                    return fmt.Errorf("Cannot append positional argument after an optional positional argument")
            }
        }
        this.posArgs = append(this.posArgs, arg)
    }else {
        this.optArgs = append(this.optArgs, arg)
    }
    return nil
}

func (this *ArgumentParser) Options() interface{} {
    return this.target
}

func (this *SingleArgument) NeedData() bool {
    if this.value.Kind() == reflect.Bool {
        return false
    }else {
        return true
    }
}

func (this *SingleArgument) MetaVar() string {
    if len(this.metavar) > 0 {
        return this.metavar
    }else if len(this.choices) > 0 {
        return fmt.Sprintf("{%s}", strings.Join(this.choices, ","))
    }else {
        return strings.ToUpper(strings.Replace(this.Token(), "-", "_", -1))
    }
}

func isUpperChar(ch byte) bool {
    return ch >= 'A' && ch <= 'Z'
}

func splitCamelString(str string) string {
    if strings.ToUpper(str) == str {
        return str
    }
    var buf bytes.Buffer
    for i := 0; i < len(str); i ++ {
        c := str[i]
        if isUpperChar(c) {
            if buf.Len() > 0 && !isUpperChar(str[i-1]) && str[i-1] != '-' {
                buf.WriteByte('-')
            }
            buf.WriteByte(c - 'A' + 'a')
        }else if c == '_' {
            buf.WriteByte('-')
        }else {
            buf.WriteByte(c)
        }
    }
    return buf.String()
}

func (this *SingleArgument) Token() string {
    return splitCamelString(this.token)
}

func (this *SingleArgument) ShortToken() string {
    return this.shortToken
}

func (this *SingleArgument) String() string {
    var start, end byte
    if this.IsOptional() {
        start = '['
        end = ']'
    }else {
        start = '<'
        end = '>'
    }
    if this.IsPositional() {
        return fmt.Sprintf("%c%s%c", start, this.MetaVar(), end)
    }else {
        if this.NeedData() {
            return fmt.Sprintf("%c--%s %s%c", start, this.Token(), this.MetaVar(), end)
        }else {
            return fmt.Sprintf("%c--%s%c", start, this.Token(), end)
        }
    }
}

func (this *SingleArgument) IsOptional() bool {
    return this.optional
}

func (this *SingleArgument) IsPositional() bool {
    return this.positional
}

func (this *SingleArgument) IsMulti() bool {
    return false
}

func (this *SingleArgument) IsSubcommand() bool {
    return false
}

func (this *SingleArgument) HelpString(indent string) string {
    return indent + strings.Join(strings.Split(this.help, "\n"), "\n" + indent)
}

func (this *SingleArgument) InChoices(val string) bool {
    if len(this.choices) > 0 {
        for _, s := range this.choices {
            if s == val {
                return true
            }
        }
        return false
    }else {
        return true
    }
}

func (this *SingleArgument) SetValue(val string) error {
    if ! this.InChoices(val)  {
        return fmt.Errorf("Unknown argument %s for %s%s", val, this.token, this.MetaVar())
    }
    e := gotypes.SetValue(this.value, val)
    if e != nil {
        return e
    }
    this.isSet = true
    return nil
}

func (this *SingleArgument) DoAction() error {
    if this.value.Type() == gotypes.BoolType {
        if this.useDefault {
            this.value.SetBool(!this.defValue.Bool())
        }else {
            this.value.SetBool(true)
        }
        this.isSet = true
    }
    return nil
}

func (this *SingleArgument) Validate() error {
    if ! this.optional && ! this.isSet && ! this.useDefault {
        return fmt.Errorf("Non-optional argument %s not set", this.token)
    }
    if ! this.isSet && this.useDefault {
        this.value.Set(this.defValue)
    }
    return nil
}

func (this *MultiArgument) IsMulti() bool {
    return true
}

func (this *MultiArgument) SetValue(val string) error {
    if ! this.InChoices(val)  {
        return fmt.Errorf("Unknown argument %s for %s%s", val, this.Token(), this.MetaVar())
    }
    var e error = nil
    e = gotypes.AppendValue(this.value, val)
    if e != nil {
        return e
    }
    this.isSet = true
    return nil
}

func (this *MultiArgument) Validate() error {
    var e = this.SingleArgument.Validate()
    if e != nil {
        return e
    }
    var vallen int64 = int64(this.value.Len())
    if (this.minCount >= 0 && vallen < this.minCount) {
        return fmt.Errorf("Argument count requires at least %d", this.minCount)
    }
    if (this.maxCount >= 0 && vallen > this.maxCount) {
        return fmt.Errorf("Argument count requires at most %d", this.maxCount)
    }
    return nil
}

func (this *SubcommandArgument) IsSubcommand() bool {
    return true
}

func (this *SubcommandArgument) String() string {
    return fmt.Sprintf("<%s>", strings.ToUpper(this.token))
}

func (this *SubcommandArgument) AddSubParser(target interface{}, command string, desc string, callback interface{}) (*ArgumentParser, error) {
    prog := fmt.Sprintf("%s %s", this.parser.prog, command)
    parser, e := NewArgumentParser(target, prog, desc, "")
    if e != nil {
        return nil, e
    }
    cbfunc := reflect.ValueOf(callback)
    this.subcommands[command] = SubcommandArgumentData{parser: parser,
                                                callback: cbfunc}
    this.choices = append(this.choices, command)
    return parser, nil
}

func (this *SubcommandArgument) HelpString(indent string) string {
    var buf bytes.Buffer
    for k, data := range this.subcommands {
        buf.WriteString(indent)
        buf.WriteString(k)
        buf.WriteByte('\n')
        buf.WriteString(indent)
        buf.WriteString("  ")
        buf.WriteString(data.parser.ShortDescription())
        buf.WriteByte('\n')
    }
    return buf.String()
}

func (this *SubcommandArgument) SubHelpString(cmd string) (string, error) {
    val, ok := this.subcommands[cmd]
    if ok {
        return val.parser.HelpString(), nil
    }else {
        return "", fmt.Errorf("No such command %s", cmd)
    }
}

func (this *SubcommandArgument) GetSubParser() *ArgumentParser {
    var cmd = this.value.String()
    val, ok := this.subcommands[cmd]
    if ok {
        return val.parser
    }else {
        return nil
    }
}

func (this *SubcommandArgument) Invoke(args ...interface{}) error {
    var inargs = make([]reflect.Value, 0)
    for _, arg := range args {
        inargs = append(inargs, reflect.ValueOf(arg))
    }
    var cmd = this.value.String()
    val, ok := this.subcommands[cmd]
    if ! ok {
        return fmt.Errorf("Unknown subcommand %s", cmd)
    }
    out := val.callback.Call(inargs)
    if len(out) == 1 {
        if out[0].IsNil() {
            return nil
        }else {
            return out[0].Interface().(error)
        }
    }else {
        return fmt.Errorf("Callback return %d unknown outputs", len(out))
    }
}

func (this *ArgumentParser) ShortDescription() string {
    return strings.Split(this.description, "\n")[0]
}

func (this *ArgumentParser) Usage() string {
    var buf bytes.Buffer
    buf.WriteString("Usage: ")
    buf.WriteString(this.prog)
    for _, arg := range this.optArgs {
        buf.WriteByte(' ')
        buf.WriteString(arg.String())
    }
    for _, arg := range this.posArgs {
        buf.WriteByte(' ')
        buf.WriteString(arg.String())
        if arg.IsSubcommand() || arg.IsMulti() {
            buf.WriteString(" ...")
        }
    }
    buf.WriteByte('\n')
    buf.WriteByte('\n')
    return buf.String()
}

func (this *ArgumentParser) HelpString() string {
    var buf bytes.Buffer
    buf.WriteString(this.Usage())
    buf.WriteString(this.description)
    buf.WriteByte('\n')
    buf.WriteByte('\n')
    if len(this.posArgs) > 0 {
        buf.WriteString("Positional arguments:\n")
        for _, arg := range this.posArgs {
            buf.WriteString("    ")
            buf.WriteString(arg.String())
            buf.WriteByte('\n')
            buf.WriteString(arg.HelpString("        "))
            buf.WriteByte('\n')
        }
        buf.WriteByte('\n')
    }
    if len(this.optArgs) > 0 {
        buf.WriteString("Optional arguments:\n")
        for _, arg := range this.optArgs {
            buf.WriteString("    ")
            buf.WriteString(arg.String())
            buf.WriteByte('\n')
            buf.WriteString(arg.HelpString("        "))
            buf.WriteByte('\n')
        }
        buf.WriteByte('\n')
    }
    if len(this.epilog) > 0 {
        buf.WriteString(this.epilog)
        buf.WriteByte('\n')
        buf.WriteByte('\n')
    }
    return buf.String()
}

func (this *ArgumentParser) findOptionalArgument(token string) Argument {
    var match_arg Argument = nil
    for _, arg := range this.optArgs {
        if strings.HasPrefix(arg.Token(), token) || strings.HasPrefix(arg.ShortToken(), token) {
            if match_arg != nil {
                return nil
            }else {
                match_arg = arg
            }
        }
    }
    return match_arg
}

func validateArgs(args []Argument) error {
    for _, arg := range args {
        e := arg.Validate()
        if e != nil {
            return fmt.Errorf("%s error: %s", arg.Token(), e)
        }
    }
    return nil
}

func (this *ArgumentParser) Validate() error {
    var e error = nil
    e = validateArgs(this.posArgs)
    if e != nil {
        return e
    }
    e = validateArgs(this.optArgs)
    if e != nil {
        return e
    }
    return nil
}

func (this *ArgumentParser) ParseArgs(args []string, ignore_unknown bool) error {
    var pos_idx int = 0
    var arg Argument = nil
    var err error = nil
    for i := 0; i < len(args); i ++ {
        if strings.HasPrefix(args[i], "-") {
            arg = this.findOptionalArgument(strings.TrimLeft(args[i], "-"))
            if arg != nil {
                if arg.NeedData() {
                    if i + 1 < len(args) {
                        err = arg.SetValue(args[i+1])
                        if err != nil {
                            return err
                        }
                        i ++
                    }else {
                        return fmt.Errorf("Missing arguments for %s", args[i])
                    }
                }else {
                    err = arg.DoAction()
                    if err != nil {
                        return err
                    }
                }
            }else if ! ignore_unknown {
                return fmt.Errorf("Unknown optional argument %s", args[i])
            }
        }else {
            if pos_idx >= len(this.posArgs) {
                if len(this.posArgs) > 0 {
                    last_arg := this.posArgs[len(this.posArgs)-1]
                    if last_arg.IsMulti() {
                        last_arg.SetValue(args[i])
                    } else if ! ignore_unknown {
                        return fmt.Errorf("Unknown positional argument %s", args[i])
                    }
                } else if ! ignore_unknown {
                    return fmt.Errorf("Unknown positional argument %s", args[i])
                }
            }else {
                arg = this.posArgs[pos_idx]
                pos_idx += 1
                err = arg.SetValue(args[i])
                if err != nil {
                    return err
                }
                if arg.IsSubcommand() {
                    var subarg *SubcommandArgument = arg.(*SubcommandArgument)
                    var subparser = subarg.GetSubParser()
                    err = subparser.ParseArgs(args[i+1:], ignore_unknown)
                    if err != nil {
                        return err
                    }
                    break
                }
            }
        }
    }
    if pos_idx < len(this.posArgs) && ! this.posArgs[pos_idx].IsOptional() {
        return fmt.Errorf("Not enough arguments")
    }
    return this.Validate()
}

func (this *ArgumentParser) parseKeyValue(key, value string) error {
    arg := this.findOptionalArgument(key)
    if arg != nil {
        return arg.SetValue(value)
    } else {
        log.Printf("Cannot found argument %s", key)
    }
    return nil
}

func (this *ArgumentParser) ParseFile(filepath string) error {
    file, e := os.Open(filepath)
    if e != nil {
        return e
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := scanner.Text()
        pos := strings.IndexByte(line, '=')
        if pos > 0 && pos < len(line) {
            key := strings.Replace(strings.Trim(line[:pos], " "), "_", "-", -1)
            val := strings.Trim(line[pos+1:], " ")
            this.parseKeyValue(key, val)
        } else {
            return fmt.Errorf("Misformated line: %s", line)
        }
    }

    if err := scanner.Err(); err != nil {
        return err
    }

    return nil
}

func (this *ArgumentParser) GetSubcommand() *SubcommandArgument {
    if len(this.posArgs) > 0 {
        last_arg := this.posArgs[len(this.posArgs)-1]
        if last_arg.IsSubcommand() {
            return last_arg.(*SubcommandArgument)
        }
    }
    return nil
}

func (this *ArgumentParser) ParseKnownArgs(args []string) error {
    return this.ParseArgs(args, true)
}

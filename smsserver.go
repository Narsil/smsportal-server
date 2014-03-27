package main

import (
    "database/sql"
    "github.com/coopernurse/gorp"
    "github.com/Narsil/xmpp"
    _ "github.com/mattn/go-sqlite3"
    "log"
    "io"
    "time"
    "net/http"
    "encoding/json"
    "fmt"
    "strings"
    "html"
    "html/template"
)

type Message struct {
    Id      int64
    Created int64
    Message    string
    From         string
    To    string
    Sent    bool
    Incoming bool
}

func authenticate(username, password string) bool{
    return true
}

func main(){
    dbmap := initDb()
    defer dbmap.Db.Close()

    // create two posts
    // dbmap.TruncateTables()
    // p1 := newMessage("+33672317534", "Hey from go")

    // // // insert rows - auto increment PKs will be set properly after the insert
    // dbmap.Insert(&p1)


    http.HandleFunc("/pending/", func(w http.ResponseWriter, r *http.Request) {
        var messages []Message
        _, err := dbmap.Select(&messages, "SELECT * FROM messages WHERE Sent=? AND Incoming=?", false, false)
        checkErr(err, "Select failed")

        str, err := json.Marshal(messages)
        checkErr(err, "Marshalling failed")

        fmt.Fprintf(w, "%s", str)
    })
    http.HandleFunc("/send/", func(w http.ResponseWriter, r *http.Request) {
        err := r.ParseForm()
        checkErr(err, "Parsing form failed")

        if (r.Form.Get("To") != "" && r.Form.Get("Message") != ""){
            message := newMessage(r.Form.Get("To"), r.Form.Get("Message"))
            err = dbmap.Insert(&message)
            checkErr(err, "Could not insert new message")
            http.Redirect(w, r, "/send/", 302)
            return
        }

        var messages []Message
        _, err = dbmap.Select(&messages, "SELECT * FROM messages ORDER BY Created DESC")
        checkErr(err, "Select failed")


        tmpl, _ := template.New("send").Parse(`
            <html>
                <body>
                    <form action="." method="POST">
                        <input type="text" name="To" placeholder="To"/>
                        <input type="text" name="Message" placeholder="Message"/>
                        <input type="submit" value="Envoyer"/>
                    </form>
                    {{range .Messages}}
                        <p style="border: 1px solid black">
                            <span style="font-weight:bold">From: {{.From}} to {{.To}}</span> {{.Message}}
                            <span style="float:right">
                                Sent: {{.Sent}}
                                Created: {{.Created}}
                                Incoming: {{.Incoming}}
                            </span>
                        </p>
                    {{end}}
                </body>
            </html>
        `)
        tmpl.Execute(w, map[string]interface{} {
            "Messages": messages,
        })

    })
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        err := r.ParseForm()
        checkErr(err, "Parsing form failed")

        if (r.Form.Get("To") != "" && r.Form.Get("Message") != ""){
            message := newMessage(r.Form.Get("To"), r.Form.Get("Message"))
            err = dbmap.Insert(&message)
            checkErr(err, "Could not insert new message")
            http.Redirect(w, r, "/", 302)
            return
        }


        tmpl, _ := template.New("homepage").Parse(`
            <!DOCTYPE html>
            <html>
                <head>
                    <link href="/static/bootstrap/css/bootstrap.min.css" type="text/css" rel="stylesheet">
                    <link href="/static/bootstrap/css/bootstrap-responsive.min.css" type="text/css" rel="stylesheet">
                    <script src="/static/jquery-1.10.2.min.js" type="text/javascript"></script>
                    <script src="/static/bootstrap/js/bootstrap.js" type="text/javascript"></script>
                    <script src="/static/sms.js" type="text/javascript"></script>
                    <meta name="viewport" content="width=device-width, initial-scale=1.0">
                </head>
                <body>
                </body>
            </html>
        `)
        tmpl.Execute(w, nil)

    })
    http.HandleFunc("/history/", func(w http.ResponseWriter, r *http.Request) {
        var messages []Message
        _, err := dbmap.Select(&messages, "SELECT * FROM messages ORDER BY Created DESC")
        checkErr(err, "Select failed")
        b, err := json.Marshal(messages)
        checkErr(err, "Encoding failed")
        fmt.Fprintf(w, "%s", b)
    })
    http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
    http.HandleFunc("/sent/", func(w http.ResponseWriter, r *http.Request) {
        err := r.ParseForm()
        checkErr(err, "Parsing form failed")

        ids_string := r.Form.Get("Id")


        ids := strings.Split(ids_string, ",")
        new_ids_string := "\"" + strings.Join(ids, "\", \"") + "\""

        query := fmt.Sprintf("UPDATE messages SET Sent='true' WHERE Id IN (%s)", new_ids_string)

        _, err = dbmap.Exec(query)

        if err == nil{
            fmt.Fprintf(w, "Ok")
        }else{
            fmt.Fprintf(w, "Error", err)
        }

    })
    srv := xmpp.NewServer()
    http.HandleFunc("/received/", func(w http.ResponseWriter, r *http.Request) {
        err := r.ParseForm()
        checkErr(err, "Parsing form failed")

        from := r.Form.Get("From")
        message := r.Form.Get("Message")

        if (from != "" && message != ""){
            m := newIncomingMessage(from, message, srv)
            err = dbmap.Insert(&m)
        }
        if err == nil{
            fmt.Fprintf(w, "Ok")
        }else{
            fmt.Fprintf(w, "Error", err)
        }

    })
    err := srv.SetKeyPair("private.cert", "private.key")
    if (err != nil){
    }
    srv.SetAuthFunc(authenticate)
    srv.SetHandleIncomingMessage(
        func (msg xmpp.XmppMessage){
            for _, body := range(msg.Bodies){
                number := strings.Split(msg.To, "@")[0]
                message := newMessage(number, html.UnescapeString(body.Body))
                err := dbmap.Insert(&message)
                checkErr(err, "Could not insert new message")
            }
    })
    srv.HandleQuery("http://jabber.org/protocol/disco#info", func (w io.Writer, iq xmpp.IQ){
            w.Write([]byte(`<iq type='result' from='sms.nicolas.kwyk.fr' to='nicolas@kwyk.fr/YY9R80gi' id='` + iq.Id + `'>
                        <query xmlns='http://jabber.org/protocol/disco#info'>
                            <feature var='http://jabber.org/protocol/disco#info'/>
                            <feature var='http://jabber.org/protocol/disco#items'/>
                        </query>
                    </iq>
            `))
    })
    srv.HandleQuery("http://jabber.org/protocol/disco#items", func (w io.Writer, iq xmpp.IQ){
        w.Write([]byte(`<iq type='result' from='sms.nicolas.kwyk.fr' to='nicolas@kwyk.fr/YY9R80gi' id='` + iq.Id + `'>
                <error code="501" type="cancel"><feature-not-implemented xmlns="urn:ietf:params:xml:ns:xmpp-stanzas"/></error>
                </iq>
        `))
    })
    srv.HandleQuery("jabber:iq:roster", func(w io.Writer, iq xmpp.IQ){
            w.Write([]byte(`<iq type='result' to='nicolas@kwyk.fr/YY9R80gi' id='` + iq.Id + `'>
                    <query xmlns="jabber:iq:roster">
                        <item jid="+33609781500@sms.nicolas.kwyk.fr" subscription="both" name="Julie Famery Phone"><group>Buddies</group></item>
                        <item jid="+33672317534@sms.nicolas.kwyk.fr" subscription="both" name="Nicolas Patry Phone"><group>Buddies</group></item>
                    </query>
                </iq>
            `))
            w.Write([]byte(`<presence from="+33609781500@sms.nicolas.kwyk.fr" to="nicolas@kwyk.fr/YY9R80gi"><caps:c node="http://www.android.com/gtalk/client/caps" ver="1.1" xmlns:caps="http://jabber.org/protocol/caps"/></presence>`))
            w.Write([]byte(`<presence from="+33672317534@sms.nicolas.kwyk.fr" to="nicolas@kwyk.fr/YY9R80gi"><caps:c node="http://www.android.com/gtalk/client/caps" ver="1.1" xmlns:caps="http://jabber.org/protocol/caps"/></presence>`))
    })


    go func(){
        srv.ListenAndServe("tcp", ":5222")
    }()
    log.Fatal(http.ListenAndServe(":8080", nil))

}


func newMessage(destination, message string) Message {
    fmt.Println("Sending SMS to: ", destination)
    return Message{
        Created: time.Now().UnixNano(),
        From: "+33672317534",
        Message:   message,
        To:    destination,
        Sent: false,
        Incoming: false,
    }
}
func newIncomingMessage(author, message string, srv xmpp.Server) Message {
    fmt.Println("Received SMS from: ", author)
    go func(){
        srv.MessageChannel<-xmpp.Message{From: author, Message:message, To: "nicolas"}
    }()

    return Message{
        Created: time.Now().UnixNano(),
        From: author,
        Message:   message,
        To:    "+33672317534",
        Sent: true,
        Incoming: true,
    }
}

func initDb() *gorp.DbMap {
    // connect to db using standard Go database/sql API
    // use whatever database/sql driver you wish
    db, err := sql.Open("sqlite3", "db.bin")
    checkErr(err, "sql.Open failed")

    // construct a gorp DbMap
    dbmap := &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}

    // add a table, setting the table name to 'posts' and
    // specifying that the Id property is an auto incrementing PK
    dbmap.AddTableWithName(Message{}, "messages").SetKeys(true, "Id")

    // create the table. in a production system you'd generally
    // use a migration tool, or create the tables via scripts
    err = dbmap.CreateTablesIfNotExists()
    checkErr(err, "Create tables failed")

    return dbmap
}

func checkErr(err error, msg string) {
    if err != nil {
        log.Fatalln(msg, err)
    }
}



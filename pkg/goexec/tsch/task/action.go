package task

import (
	"encoding/xml"
)

// ---------------------------------------------------------------------------
// shared base
// ---------------------------------------------------------------------------

// ActionType is the base for all actions (only carries the optional id attribute).
type ActionType struct {
	XMLName xml.Name `xml:"-"`
	Id      string   `xml:"id,attr,omitempty"`
}

// ---------------------------------------------------------------------------
// Exec
// ---------------------------------------------------------------------------

// ExecAction corresponds to <Exec> (execActionType).
type ExecAction struct {
	XMLName xml.Name `xml:"Exec"`
	ActionType

	// <Command> is the program or script to run.
	Command string `xml:"Command"`
	// <Arguments> are passed to the Command.
	Arguments string `xml:"Arguments,omitempty"`
	// <WorkingDirectory> sets the cwd for the process.
	WorkingDirectory string `xml:"WorkingDirectory,omitempty"`
}

// ---------------------------------------------------------------------------
// ComHandler
// ---------------------------------------------------------------------------

// ComHandlerAction corresponds to <ComHandler> (comHandlerActionType).
type ComHandlerAction struct {
	XMLName xml.Name `xml:"ComHandler"`
	ActionType

	// <ClassId> is the COM class ID (GUID).
	ClassId string `xml:"ClassId"`
	// <Data> is passed into the handler (optional).
	Data string `xml:"Data,omitempty"`
}

// ---------------------------------------------------------------------------
// SendEmail
// ---------------------------------------------------------------------------

// SendEmailAction corresponds to <SendEmail> (sendEmailActionType).
type SendEmailAction struct {
	XMLName xml.Name `xml:"SendEmail"`
	ActionType

	Server  string `xml:"Server"`  // SMTP server
	Subject string `xml:"Subject"` // email subject
	To      string `xml:"To"`      // semicolon‑separated
	Cc      string `xml:"Cc,omitempty"`
	Bcc     string `xml:"Bcc,omitempty"`
	ReplyTo string `xml:"ReplyTo,omitempty"`
	Body    string `xml:"Body,omitempty"`
	// optional named header fields
	HeaderFields *NamedValues `xml:"HeaderFields,omitempty"`
}

// ---------------------------------------------------------------------------
// ShowMessage
// ---------------------------------------------------------------------------

// ShowMessageAction corresponds to <ShowMessage> (showMessageActionType).
type ShowMessageAction struct {
	XMLName xml.Name `xml:"ShowMessage"`
	ActionType

	Title   string `xml:"Title"`   // window title
	Message string `xml:"Message"` // body text
}

// ---------------------------------------------------------------------------
// NamedValues (used by SendEmailAction.HeaderFields)
// ---------------------------------------------------------------------------

// NamedValues holds zero or more <Value name="…">…</Value> entries.
type NamedValues struct {
	XMLName xml.Name     //`xml:"HeaderFields"`
	Value   []NamedValue `xml:"Value"`
}

// NamedValue is one name/value pair.
type NamedValue struct {
	XMLName xml.Name `xml:"Value"`
	Name    string   `xml:"name,attr"`
	Value   string   `xml:",chardata"`
}

// ---------------------------------------------------------------------------
// Actions container
// ---------------------------------------------------------------------------

// Actions corresponds to <Actions> (actionsType).
// It may contain any number of each action type, in any order,
// and carries an optional Context attribute.
type Actions struct {
	XMLName xml.Name `xml:"Actions"`

	// Context="" lets you override the default ("Author").
	Context string `xml:"Context,attr,omitempty"`

	Exec        []ExecAction        `xml:"Exec,omitempty"`
	ComHandler  []ComHandlerAction  `xml:"ComHandler,omitempty"`
	SendEmail   []SendEmailAction   `xml:"SendEmail,omitempty"`
	ShowMessage []ShowMessageAction `xml:"ShowMessage,omitempty"`
}

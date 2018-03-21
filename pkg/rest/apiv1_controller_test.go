package rest

import (
	"encoding/json"
	"io"
	"net/mail"
	"net/textproto"
	"os"
	"testing"
	"time"

	"github.com/jhillyerd/enmime"
	"github.com/jhillyerd/inbucket/pkg/message"
	"github.com/jhillyerd/inbucket/pkg/test"
)

const (
	baseURL = "http://localhost/api/v1"

	// JSON map keys
	mailboxKey = "mailbox"
	idKey      = "id"
	fromKey    = "from"
	toKey      = "to"
	subjectKey = "subject"
	dateKey    = "date"
	sizeKey    = "size"
	headerKey  = "header"
	bodyKey    = "body"
	textKey    = "text"
	htmlKey    = "html"
)

func TestRestMailboxList(t *testing.T) {
	// Setup
	mm := test.NewManager()
	logbuf := setupWebServer(mm)

	// Test invalid mailbox name
	w, err := testRestGet(baseURL + "/mailbox/foo@bar")
	expectCode := 500
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Errorf("Expected code %v, got %v", expectCode, w.Code)
	}

	// Test empty mailbox
	w, err = testRestGet(baseURL + "/mailbox/empty")
	expectCode = 200
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Errorf("Expected code %v, got %v", expectCode, w.Code)
	}

	// Test Mailbox error
	w, err = testRestGet(baseURL + "/mailbox/messageserr")
	expectCode = 500
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Errorf("Expected code %v, got %v", expectCode, w.Code)
	}

	// Test JSON message headers
	tzPDT := time.FixedZone("PDT", -7*3600)
	tzPST := time.FixedZone("PST", -8*3600)
	meta1 := message.Metadata{
		Mailbox: "good",
		ID:      "0001",
		From:    &mail.Address{Name: "", Address: "from1@host"},
		To:      []*mail.Address{{Name: "", Address: "to1@host"}},
		Subject: "subject 1",
		Date:    time.Date(2012, 2, 1, 10, 11, 12, 253, tzPST),
	}
	meta2 := message.Metadata{
		Mailbox: "good",
		ID:      "0002",
		From:    &mail.Address{Name: "", Address: "from2@host"},
		To:      []*mail.Address{{Name: "", Address: "to1@host"}},
		Subject: "subject 2",
		Date:    time.Date(2012, 7, 1, 10, 11, 12, 253, tzPDT),
	}
	mm.AddMessage("good", &message.Message{Metadata: meta1})
	mm.AddMessage("good", &message.Message{Metadata: meta2})

	// Check return code
	w, err = testRestGet(baseURL + "/mailbox/good")
	expectCode = 200
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Fatalf("Expected code %v, got %v", expectCode, w.Code)
	}

	// Check JSON
	dec := json.NewDecoder(w.Body)
	var result []interface{}
	if err := dec.Decode(&result); err != nil {
		t.Errorf("Failed to decode JSON: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("Expected 2 results, got %v", len(result))
	}

	decodedStringEquals(t, result, "[0]/mailbox", "good")
	decodedStringEquals(t, result, "[0]/id", "0001")
	decodedStringEquals(t, result, "[0]/from", "<from1@host>")
	decodedStringEquals(t, result, "[0]/to/[0]", "<to1@host>")
	decodedStringEquals(t, result, "[0]/subject", "subject 1")
	decodedStringEquals(t, result, "[0]/date", "2012-02-01T10:11:12.000000253-08:00")
	decodedNumberEquals(t, result, "[0]/size", 0)
	decodedStringEquals(t, result, "[1]/mailbox", "good")
	decodedStringEquals(t, result, "[1]/id", "0002")
	decodedStringEquals(t, result, "[1]/from", "<from2@host>")
	decodedStringEquals(t, result, "[1]/to/[0]", "<to1@host>")
	decodedStringEquals(t, result, "[1]/subject", "subject 2")
	decodedStringEquals(t, result, "[1]/date", "2012-07-01T10:11:12.000000253-07:00")
	decodedNumberEquals(t, result, "[1]/size", 0)

	if t.Failed() {
		// Wait for handler to finish logging
		time.Sleep(2 * time.Second)
		// Dump buffered log data if there was a failure
		_, _ = io.Copy(os.Stderr, logbuf)
	}
}

func TestRestMessage(t *testing.T) {
	// Setup
	mm := test.NewManager()
	logbuf := setupWebServer(mm)

	// Test invalid mailbox name
	w, err := testRestGet(baseURL + "/mailbox/foo@bar/0001")
	expectCode := 500
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Errorf("Expected code %v, got %v", expectCode, w.Code)
	}

	// Test requesting a message that does not exist
	w, err = testRestGet(baseURL + "/mailbox/empty/0001")
	expectCode = 404
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Errorf("Expected code %v, got %v", expectCode, w.Code)
	}

	// Test GetMessage error
	w, err = testRestGet(baseURL + "/mailbox/messageerr/0001")
	expectCode = 500
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Errorf("Expected code %v, got %v", expectCode, w.Code)
	}

	if t.Failed() {
		// Wait for handler to finish logging
		time.Sleep(2 * time.Second)
		// Dump buffered log data if there was a failure
		_, _ = io.Copy(os.Stderr, logbuf)
	}

	// Test JSON message headers
	tzPST := time.FixedZone("PST", -8*3600)
	msg1 := message.New(
		message.Metadata{
			Mailbox: "good",
			ID:      "0001",
			From:    &mail.Address{Name: "", Address: "from1@host"},
			To:      []*mail.Address{{Name: "", Address: "to1@host"}},
			Subject: "subject 1",
			Date:    time.Date(2012, 2, 1, 10, 11, 12, 253, tzPST),
		},
		&enmime.Envelope{
			Text: "This is some text",
			HTML: "This is some HTML",
			Root: &enmime.Part{
				Header: textproto.MIMEHeader{
					"To":   []string{"fred@fish.com", "keyword@nsa.gov"},
					"From": []string{"noreply@inbucket.org"},
				},
			},
		},
	)
	mm.AddMessage("good", msg1)

	// Check return code
	w, err = testRestGet(baseURL + "/mailbox/good/0001")
	expectCode = 200
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Fatalf("Expected code %v, got %v", expectCode, w.Code)
	}

	// Check JSON
	dec := json.NewDecoder(w.Body)
	var result map[string]interface{}
	if err := dec.Decode(&result); err != nil {
		t.Errorf("Failed to decode JSON: %v", err)
	}

	decodedStringEquals(t, result, "mailbox", "good")
	decodedStringEquals(t, result, "id", "0001")
	decodedStringEquals(t, result, "from", "<from1@host>")
	decodedStringEquals(t, result, "to/[0]", "<to1@host>")
	decodedStringEquals(t, result, "subject", "subject 1")
	decodedStringEquals(t, result, "date", "2012-02-01T10:11:12.000000253-08:00")
	decodedNumberEquals(t, result, "size", 0)
	decodedStringEquals(t, result, "body/text", "This is some text")
	decodedStringEquals(t, result, "body/html", "This is some HTML")
	decodedStringEquals(t, result, "header/To/[0]", "fred@fish.com")
	decodedStringEquals(t, result, "header/To/[1]", "keyword@nsa.gov")
	decodedStringEquals(t, result, "header/From/[0]", "noreply@inbucket.org")

	if t.Failed() {
		// Wait for handler to finish logging
		time.Sleep(2 * time.Second)
		// Dump buffered log data if there was a failure
		_, _ = io.Copy(os.Stderr, logbuf)
	}
}

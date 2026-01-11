package pm5

import (
	"context"
	"encoding/hex"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/seagrayinc/pm5-csafe/pkg/csafe"
	"github.com/seagrayinc/pm5-csafe/pkg/hid"
)

// parseHexString converts a dash-separated hex string to bytes
func parseHexString(s string) []byte {
	// Remove dashes
	s = strings.ReplaceAll(s, "-", "")
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

// TestEndToEnd exercises the full end-to-end flow:
// 1. Raw HID report bytes are received (simulated via MockHID)
// 2. CSAFE frame parsing extracts the frame from the report
// 3. Command response parsing extracts the expected responses
func TestEndToEnd(t *testing.T) {
	tests := []struct {
		name     string
		rawHex   string
		reportID byte
		expected []interface{}
	}{
		{
			// Frame structure (from "unstuffed bytes"=00-fd-01-92-05-30-30-30-30-30-a6):
			//   00    - Destination Address (PC Host Primary)
			//   fd    - Source Address (Default Secondary)
			//   01    - Status byte
			//   92    - Command (CSAFE_GETID_CMD)
			//   05    - Data byte count (5 bytes)
			//   30-30-30-30-30 - Data (ASCII "00000")
			//   a6    - Checksum
			name:     "GetIDResponse",
			rawHex:   "f0-00-fd-01-92-05-30-30-30-30-30-a6-f2-00-b3-f2-00-00-00-00-00-00-09-00-b4-03-07-00-58-e6-f2-f2-00-35-29-00-00-00-00-00-00-00-30-00-81-f2-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00",
			reportID: 0x02,
			expected: []interface{}{
				GetIDResponse{
					ASCIIDigit0: '0',
					ASCIIDigit1: '0',
					ASCIIDigit2: '0',
					ASCIIDigit3: '0',
					ASCIIDigit4: '0',
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawBytes := parseHexString(tt.rawHex)

			// Setup mock HID
			mockHID := hid.NewMockHID()

			// Create transport with mock device
			transport := &csafe.Transport{
				Device:        mockHID,
				ReportLengths: ReportLengths,
			}

			// Create context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			// Start polling for reports
			reportChan := mockHID.PollReports(ctx)
			frameChan := transport.Poll(ctx, reportChan)

			// Emit the raw bytes as a HID report
			go func() {
				mockHID.Emit(hid.Report{
					ID:   tt.reportID,
					Data: rawBytes,
				})
			}()

			// Wait for frame to be parsed
			select {
			case frame := <-frameChan:
				// Parse the responses
				responses, err := ParseResponses(frame)
				if err != nil {
					t.Fatalf("ParseResponses failed: %v", err)
				}

				if len(responses) != len(tt.expected) {
					t.Fatalf("response count mismatch: got %d, want %d", len(responses), len(tt.expected))
				}

				for i, got := range responses {
					if !reflect.DeepEqual(got, tt.expected[i]) {
						t.Errorf("response[%d] mismatch:\ngot:  %+v\nwant: %+v", i, got, tt.expected[i])
					}
				}

			case <-ctx.Done():
				t.Fatal("timeout waiting for frame")
			}
		})
	}
}

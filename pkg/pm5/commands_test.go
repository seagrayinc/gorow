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
		{
			// Wrapped command response: GetWorkoutStateResponse
			// Frame: 00-fd-01-1a-03-8d-01-01-95
			//   1a    - CSAFE_SETUSERCFG1_CMD (wrapper)
			//   03    - wrapper data length
			//   8d    - CSAFE_PM_GET_WORKOUTSTATE
			//   01    - data length
			//   01    - WorkoutState (1 = Workout row)
			name:     "GetWorkoutStateResponse",
			rawHex:   "f0-00-fd-01-1a-03-8d-01-01-95-f2-8d-01-01-30-f2-00-00-00-00-00-00-09-00-b4-03-07-00-58-e6-f2-f2-00-35-29-00-00-00-00-00-00-00-30-00-81-f2-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00",
			reportID: 0x02,
			expected: []interface{}{
				GetWorkoutStateResponse{
					WorkoutState:       1,
					WorkoutStateString: "Workout row",
				},
			},
		},
		{
			// Multiple responses: GetStrokeStatsResponse + GetPowerResponse
			// Frame: 00-fd-81-1a-12-6e-10-b9-00-00-2a-00-38-02-00-00-00-00-00-00-00-08-00-b4-03-06-00-58-bf
			//   1a 12 - CSAFE_SETUSERCFG1_CMD wrapper with 18 bytes
			//   6e 10 - CSAFE_PM_GET_STROKESTATS with 16 bytes of data
			//   b4 03 - CSAFE_GETPOWER_CMD with 3 bytes of data
			name:     "GetStrokeStatsResponse_and_GetPowerResponse",
			rawHex:   "f0-00-fd-81-1a-12-6e-10-b9-00-00-2a-00-38-02-00-00-00-00-00-00-00-08-00-b4-03-06-00-58-bf-f2-f2-00-35-29-00-00-00-00-00-00-00-30-00-81-f2-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00",
			reportID: 0x02,
			expected: []interface{}{
				GetStrokeStatsResponse{
					StrokeDistance:     185,
					StrokeDriveTime:    0,
					StrokeRecoveryTime: 42,
					StrokeLength:       56,
					DriveCounter:       2,
					PeakDriveForce:     0,
					ImpulseDriveForce:  0,
					AverageDriveForce:  0,
					WorkPerStroke:      8,
				},
				GetPowerResponse{
					StrokeWatts:    6,
					UnitsSpecifier: 88,
				},
			},
		},
		{
			// Synthetic test case to exercise byte unstuffing
			// Unstuffed frame: 00-fd-01-92-05-f0-f1-f2-f3-30-checksum
			// The data bytes f0-f1-f2-f3-30 represent ASCII values where:
			//   f0 (240), f1 (241), f2 (242), f3 (243), 30 ('0')
			// Checksum calculation: 01 ^ 92 ^ 05 ^ f0 ^ f1 ^ f2 ^ f3 ^ 30 = 0xA6
			// After byte stuffing: f0-f1-f2-f3-30 becomes f3-00-f3-01-f3-02-f3-03-30
			// Full stuffed frame: 00-fd-01-92-05-f3-00-f3-01-f3-02-f3-03-30-a6
			name:     "GetIDResponse_with_byte_stuffing",
			rawHex:   "f0-00-fd-01-92-05-f3-00-f3-01-f3-02-f3-03-30-a6-f2-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00-00",
			reportID: 0x02,
			expected: []interface{}{
				GetIDResponse{
					ASCIIDigit0: 0xF0,
					ASCIIDigit1: 0xF1,
					ASCIIDigit2: 0xF2,
					ASCIIDigit3: 0xF3,
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

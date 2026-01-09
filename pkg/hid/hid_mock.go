package hid

import (
	"context"
)

type MockHID struct {
	reports chan Report
}

func NewMockHID() *MockHID {
	return &MockHID{
		reports: make(chan Report),
	}
}

func (m *MockHID) Close() error {
	return nil
}

func (m *MockHID) WriteReport(_ context.Context, _ Report) error {
	return nil
}

func (m *MockHID) PollReports(ctx context.Context) <-chan Report {
	go func() {
		<-ctx.Done()
		close(m.reports)
	}()

	return m.reports
}

func (m *MockHID) Emit(r Report) {
	//b := make([]byte, 501)
	m.reports <- Report{ID: r.ID, Data: r.Data}
}

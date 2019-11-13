package financialimporter

import "sync"

type sqlRecords struct {
	records    []map[string]string
	RecordsMux sync.Mutex
}

func NewSqlRecords() *sqlRecords {
	return &sqlRecords{records: make([]map[string]string, 0)}
}

func (s *sqlRecords) add(record map[string]string) {
	s.RecordsMux.Lock()
	defer s.RecordsMux.Unlock()
	s.records = append(s.records, record)
}

func (s *sqlRecords) borrowRecords(record map[string]string) []map[string]string {
	s.RecordsMux.Lock()
	return s.records
}

func (s *sqlRecords) returnRecords() {
	s.RecordsMux.Unlock()
}

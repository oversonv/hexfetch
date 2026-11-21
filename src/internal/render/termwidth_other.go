//go:build !linux

package render

// TermWidth возвращает 0 на платформах без реализации определения
// ширины: вывод считается неограниченным по ширине.
func TermWidth() int { return 0 }

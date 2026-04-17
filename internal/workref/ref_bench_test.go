package workref

import "testing"

func benchmarkConstructor(
	b *testing.B,
	constructor func(string, string),
) {
	b.Helper()

	for _, tc := range constructorCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				constructor(tc.id, tc.value)
			}
		})
	}
}

func runBenchmarkSuite(
	b *testing.B,
	name string,
	constructor func(string, string),
) {
	b.Helper()

	b.Run(name, func(b *testing.B) {
		benchmarkConstructor(b, constructor)
	})
}

func BenchmarkConstructors(b *testing.B) {
	runBenchmarkSuite(b, "NewPath", func(id string, value string) {
		_ = NewPath(id, value)
	})

	runBenchmarkSuite(b, "NewRoot", func(id string, value string) {
		_ = NewRoot(id, value)
	})
}

package log

import (
	"bytes"
	"reflect"
	"testing"
)

func TestNewColor(t *testing.T) {
	type args struct {
		codes []int
	}
	tests := []struct {
		name string
		args args
		want *Color
	}{
		{
			name: "no codes",
			args: args{
				codes: []int{},
			},
			want: &Color{
				codes:  nil,
				prefix: nil,
				reset:  nil,
			},
		},
		{
			name: "nil",
			args: args{
				codes: nil,
			},
			want: &Color{
				codes:  nil,
				prefix: nil,
				reset:  nil,
			},
		},
		{
			name: "single code",
			args: args{
				codes: []int{31},
			},
			want: &Color{
				codes:  []int{31},
				prefix: []byte("\x1b[31m"),
				reset:  []byte("\x1b[0m"),
			},
		},
		{
			name: "multiple codes",
			args: args{
				codes: []int{1, 34, 45},
			},
			want: &Color{
				codes:  []int{1, 34, 45},
				prefix: []byte("\x1b[1;34;45m"),
				reset:  []byte("\x1b[0m"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewColor(tt.args.codes...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewColor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestColor_WriteString(t *testing.T) {
	type fields struct {
		codes  []int
		prefix []byte
		reset  []byte
	}
	type args struct {
		buf *bytes.Buffer
		s   string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "basic",
			fields: fields{
				codes:  []int{31},
				prefix: []byte("\x1b[31m"),
				reset:  []byte("\x1b[0m"),
			},
			args: args{
				buf: &bytes.Buffer{},
				s:   "hello",
			},
			want: "\x1b[31mhello\x1b[0m",
		},
		{
			name: "no color",
			fields: fields{
				prefix: nil,
				reset:  nil,
			},
			args: args{
				buf: &bytes.Buffer{},
				s:   "hello",
			},
			want: "hello",
		},
		{
			name: "empty string",
			fields: fields{
				prefix: []byte("P"),
				reset:  []byte("R"),
			},
			args: args{
				buf: &bytes.Buffer{},
				s:   "",
			},
			want: "PR",
		},
		{
			name: "nil receiver",
			args: args{
				buf: &bytes.Buffer{},
				s:   "hello",
			},
			want: "hello",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Color{
				codes:  tt.fields.codes,
				prefix: tt.fields.prefix,
				reset:  tt.fields.reset,
			}
			c.WriteString(tt.args.buf, tt.args.s)
			if got := tt.args.buf.String(); got != tt.want {
				t.Errorf("Color.WriteString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestColor_WriteBytes(t *testing.T) {
	type fields struct {
		codes  []int
		prefix []byte
		reset  []byte
	}
	type args struct {
		buf *bytes.Buffer
		b   []byte
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "basic",
			fields: fields{
				codes:  []int{32},
				prefix: []byte("\x1b[32m"),
				reset:  []byte("\x1b[0m"),
			},
			args: args{
				buf: &bytes.Buffer{},
				b:   []byte("hello"),
			},
			want: "\x1b[32mhello\x1b[0m",
		},
		{
			name: "no color",
			fields: fields{
				prefix: nil,
				reset:  nil,
			},
			args: args{
				buf: &bytes.Buffer{},
				b:   []byte("hello"),
			},
			want: "hello",
		},
		{
			name: "nil receiver",
			args: args{
				buf: &bytes.Buffer{},
				b:   []byte("hello"),
			},
			want: "hello",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Color{
				codes:  tt.fields.codes,
				prefix: tt.fields.prefix,
				reset:  tt.fields.reset,
			}
			c.WriteBytes(tt.args.buf, tt.args.b)
			if got := tt.args.buf.String(); got != tt.want {
				t.Errorf("Color.WriteBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_makeSGR(t *testing.T) {
	type args struct {
		codes []int
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "color red",
			args: args{codes: []int{31}},
			want: []byte("\x1b[31m"),
		},
		{
			name: "bold red",
			args: args{codes: []int{1, 31}},
			want: []byte("\x1b[1;31m"),
		},
		{
			name: "empty",
			args: args{codes: nil},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := makeSGR(tt.args.codes); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("makeSGR() = %v, want %v", got, tt.want)
			}
		})
	}
}

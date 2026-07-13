package detector

import "testing"

func TestBasicTrigger(t *testing.T) {
	d := New(3, 50)
	var hit bool
	var matched string
	for _, l := range []string{"a", "b", "a", "c", "a"} {
		if m, _, h := d.Feed(l); h {
			hit = true
			matched = m
		}
	}
	if !hit {
		t.Fatal("expected a trigger after 'a' repeated 3 times")
	}
	if matched != "a" {
		t.Fatalf("expected matched line 'a', got %q", matched)
	}
}

func TestBelowThresholdDoesNotTrigger(t *testing.T) {
	d := New(3, 50)
	for _, l := range []string{"a", "b", "a", "c"} {
		if _, _, h := d.Feed(l); h {
			t.Fatal("did not expect a trigger, 'a' only repeated twice")
		}
	}
}

func TestConsecutiveDuplicatesDoNotCount(t *testing.T) {
	// Simulates a redrawn spinner: the exact same line, many times in a row.
	d := New(3, 50)
	for i := 0; i < 20; i++ {
		if _, _, h := d.Feed("Thinking..."); h {
			t.Fatal("consecutive identical lines (a spinner) should never trigger a loop")
		}
	}
}

func TestANSIStripping(t *testing.T) {
	got := Normalize("\x1b[32mHello\x1b[0m World\r")
	want := "Hello World"
	if got != want {
		t.Fatalf("Normalize() = %q, want %q", got, want)
	}
}

func TestFiresOncePerEpisode(t *testing.T) {
	d := New(3, 50)
	fires := 0
	// 'a' shows up 5 times total, separated by other lines — should still
	// only report the episode once, not once per additional occurrence.
	for _, l := range []string{"a", "b", "a", "c", "a", "d", "a", "e", "a"} {
		if _, _, h := d.Feed(l); h {
			fires++
		}
	}
	if fires != 1 {
		t.Fatalf("expected exactly 1 trigger for a sustained repeat, got %d", fires)
	}
}

func TestCanRefireAfterAgingOut(t *testing.T) {
	d := New(2, 3) // small window so "a" ages out quickly
	d.Feed("a")
	d.Feed("b")
	_, _, first := d.Feed("a") // history=[a,b,a], count(a)=2 -> fires
	if !first {
		t.Fatal("expected first episode to fire")
	}
	// Push "a" out of the window with unrelated lines so its alert flag clears.
	d.Feed("c") // history=[b,a,c]
	d.Feed("d") // history=[a,c,d]
	d.Feed("e") // history=[c,d,e] -- "a" has now fully aged out

	// A fresh episode of "a", spaced out so it isn't mistaken for a spinner redraw.
	d.Feed("a")                 // history=[d,e,a]
	d.Feed("f")                 // history=[e,a,f]
	_, _, second := d.Feed("a") // history=[a,f,a], count(a)=2 -> should fire again
	if !second {
		t.Fatal("expected a new episode of 'a' to be able to fire again after aging out of the window")
	}
}

package main

func main() {

	type Buff struct{ Bagel bool }

	Log.Info("foo", struct {
		Foo  string
		Butt Buff
	}{"bar", Buff{
		Bagel: true,
	},
	})

}

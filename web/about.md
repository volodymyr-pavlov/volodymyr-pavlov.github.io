# About me

I'm a software engineer, and I've spent the last decade building backend systems in Go. Most of that work has been in
places where the hot path can't be slow and the numbers can't be wrong: money or inventory that must not be
double-counted.

## The Leadership Side
I also lead the teams that build these systems: hiring, code review, and architecture in UML and C4. As a lead I set the
quality bar and keep it under deadline pressure.

## How I Write Code
I like code that doesn't allocate: `sync.Pool` for transient buffers, `[]byte` instead of `string` while the data is in
flight, fixed arrays `[N]T` to stay on the stack, and bitmasks instead of a struct of booleans. I
[wrote about that last one](blog/01-bitwise-in-go.html).

## Get in Touch
The best way to reach me is through the [contact page](contact.html) or on
[LinkedIn](https://www.linkedin.com/in/vladymyrpavlov/). Code lives on [GitHub](https://github.com/grakholsky).

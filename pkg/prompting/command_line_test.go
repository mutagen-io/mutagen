package prompting

// TODO: Figure out a way to test command line prompting. I'm not sure that this
// is even possible. We might be able to swap out the OS interfaces that gopass
// is calling down to, but it would be tricky in a parallel testing environment.
// Alternatively, we could restructure the function to more easily test parts of
// it, but that's just a bit pointless. The gopass package is so well-tested,
// including with tests we wrote, and the wrapper function is so simple, that I
// think we probably don't need to bother.

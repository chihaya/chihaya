## Code conventions

The project follows idiomatic [Go conventions] for style.

[Go conventions]: https://github.com/jzelinskie/conventions

## Communication

Currently, real time conversation happens on [#chihaya] on [freenode].
We are currently attempting to have more information available on GitHub.

[#chihaya]: http://webchat.freenode.net?channels=chihaya
[freenode]: http://freenode.net

## Pull request procedure

Please don't write massive patches without prior communication, as it will most
likely lead to confusion and time wasted for everyone. However, small
unannounced fixes are always welcome!

Pull requests should be targeted at the `master` branch. Before pushing to your
Github repo and issuing the pull request, please do two things:

1. [Rebase](http://git-scm.com/book/en/Git-Branching-Rebasing) your
   local changes against the `master` branch. Resolve any conflicts
   that arise.

2. Run the test suite with the `godep go test -v ./...` command.

Pull requests will be treated as "review requests", and we will give
feedback we expect to see corrected on [style] and substance before pulling.
Changes contributed via pull request should focus on a single issue at a time,
like any other. We will not accept pull-requests that try to "sneak" unrelated
changes in.

Any new files should include the license header found at the top of every
source file.

[style]: https://github.com/jzelinskie/conventions

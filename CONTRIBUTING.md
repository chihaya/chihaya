## Communication

Currently, real time conversation happens on [#chihaya] on [freenode].
We are currently attempting to have more information available on GitHub.

[#chihaya]: http://webchat.freenode.net?channels=chihaya
[freenode]: http://freenode.net

## Pull request procedure

Please don't write massive patches without prior communication, as it will most
likely lead to confusion and time wasted for everyone. However, small
unannounced fixes are always welcome!

Pull requests will be treated as "review requests", and we will give
feedback we expect to see corrected on [style] and substance before merging.
Changes contributed via pull request should focus on a single issue at a time,
like any other. We will not accept pull-requests that try to "sneak" unrelated
changes in.

The average contribution flow is as follows:

- Create a topic branch from where you want to base your work. This is usually master.
- Make commits of logical units.
- Make sure your commit messages are in the [proper format]
- Push your changes to a topic branch in your fork of the repository.
- Submit a pull request.
- Your PR will be reviewed and merged by one of the maintainers.


Any new files should include the license header found at the top of every
source file.

[style]: https://github.com/chihaya/chihaya/blob/master/CONTRIBUTING.md#style
[proper format]: https://github.com/chihaya/chihaya/blob/master/CONTRIBUTING.md#commit-messages

## Style

### Go

The project follows idiomatic [Go conventions] for style. If you're just
starting out writing Go, you can check out this [meta-package] that documents
style idiomatic style decisions you will find in open source Go code.


[Go conventions]: https://github.com/golang/go/wiki/CodeReviewComments
[meta-package]: https://github.com/jzelinskie/conventions

### Commit Messages

We follow a rough convention for commit messages that is designed to answer two
questions: what changed and why. The subject line should feature the what and
the body of the commit should describe the why.

```
scripts: add the test-cluster command

this uses tmux to setup a test cluster that you can easily kill and
start for debugging.

Fixes #38
```

The format can be described more formally as follows:

```
<subsystem>: <what changed>
<BLANK LINE>
<why this change was made>
<BLANK LINE>
<footer>
```

The first line is the subject and should be no longer than 70 characters, the
second line is always blank, and other lines should be wrapped at 80 characters.
This allows the message to be easier to read on GitHub as well as in various
git tools.

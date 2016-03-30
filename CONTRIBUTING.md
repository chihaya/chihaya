## Discussion

Long-term discussion and bug reports are maintained via [GitHub Issues].
Code review is done via [GitHub Pull Requests].
Real-time discussion is done via [freenode IRC].

[GitHub Issues]: https://github.com/chihaya/chihaya/issues
[GitHub Pull Requests]: https://github.com/chihaya/chihaya/pulls
[freenode IRC]: http://webchat.freenode.net/?channels=chihaya

## Pull Request Procedure

If you're looking to contribute, search the GitHub for issues labeled "low-hanging fruit".
You can also hop into IRC and ask a developer who's online for their opinion.

Small, self-describing fixes are perfectly fine to submit without discussion.
However, please do not submit a massive Pull Request without prior communication.
Large, unannounced changes usually lead to confusion and time wasted for everyone.
If you were planning to write a large change, post an issue on GitHub first and discuss it.

Pull Requests will be treated as "review requests", and we will give feedback we expect to see corrected on style and substance before merging.
Changes contributed via Pull Request should focus on a single issue at a time.
We will not accept pull-requests that try to "sneak" unrelated changes in.

The average contribution flow is as follows:

- Determine what to work on via creating and issue or finding an issue you want to solve.
- Create a topic branch from where you want to base your work. This is usually `master`.
- Make commits of logical units.
- Make sure your commit messages are in the proper format
- Push your changes to a topic branch in your fork of the repository.
- Submit a pull request.
- Your PR will be reviewed and merged by one of the maintainers.
- You may be asked to make changes and [rebase] your commits.

[rebase]: https://git-scm.com/book/en/v2/Git-Branching-Rebasin://git-scm.com/book/en/v2/Git-Branching-Rebasing

## Style

Any new files should include the license header found at the top of every source file.

### Go

The project follows idiomatic [Go conventions] for style.
If you're just starting out writing Go, you can check out this [meta-package] that documents style idiomatic style decisions you will find in open source Go code.
All files should have `gofmt` executed on them and code should strive to have full coverage of static analysis tools like [govet] and [golint].

[Go conventions]: https://github.com/golang/go/wiki/CodeReviewComments
[meta-package]: https://github.com/jzelinskie/conventions
[govet]: https://golang.org/cmd/vet
[golint]: https://github.com/golang/lint

### Commit Messages

We follow a rough convention for commit messages that is designed to answer two questions: what changed and why.
The subject line should feature the what and the body of the commit should describe the why.

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

The first line is the subject and should be no longer than 70 characters, the second line is always blank, and other lines should be wrapped at 80 characters.
This allows the message to be easier to read on GitHub as well as in various git tools.

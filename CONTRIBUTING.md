## Contributing

First off, thank you for considering contributing to gollum!

### 1. Where do I go from here?

If you've noticed a bug or have a question that doesn't traceable in the
[issue tracker search](https://github.com/trivago/gollum/issues?q=something)
go ahead and [make one](https://github.com/trivago/gollum/issues/new)!

### 2. Fork & create a branch

If this is something you think you can fix, then
[fork gollum](https://help.github.com/articles/fork-a-repo)
and create a branch with a descriptive name.

A good branch name would be (where issue #325 is the ticket you're working on):

```sh
git checkout -b 325-add-my-awesome-plugin
```

### 3. Get the test suite running

Make sure you're using a recent golang version and have `make` installed.

You should be able to run the entire suite using:

```sh
make test
```

The test run will execute `go vet`, `golint`, `go fmt` checks, `unit- and basic integration` tests.

### 4. Did you find a bug?

* **Ensure the bug was not already reported** by [searching all
  issues](https://github.com/trivago/gollum/issues?q=).

* If you're unable to find an open issue addressing the problem, [open a new
  one](https://github.com/trivago/gollum/issues/new).  Be sure to
  include a **title and clear description**, as much relevant information as
  possible, and a **config sample** or an **executable test case** demonstrating
  the expected behavior that is not occurring.

### 5. Implement your fix or feature

At this point, you're ready to make your changes! Feel free to ask for help;
everyone is a beginner at first :smile_cat:

### 6. Make a Pull Request

At this point, you should switch back to your master branch and make sure it's
up to date with gollum's master branch:

```sh
git remote add upstream git@github.com:trivago/gollum.git
git checkout master
git pull upstream master
```

Then update your feature branch from your local copy of master, and push it!

```sh
git checkout 325-add-my-awesome-plugin
git rebase master
git push --set-upstream origin 325-add-my-awesome-plugin
```

Finally, go to GitHub and
[make a Pull Request](https://help.github.com/articles/creating-a-pull-request)
:D

Travis CI will run our test suite against all supported golang versions. We care
about quality, so your PR won't be merged until all tests pass. It's unlikely,
but it's possible that your changes pass tests in one golang version but fail in
another. In that case, you'll have to setup your development environment (as
explained in step 3) to use the problematic golang version, and investigate
what's going on!

### 7. Keeping your Pull Request updated

If a maintainer asks you to "rebase" or "reintegrate" your PR, they're saying that a lot of code
has changed, and that you need to update your branch so it's easier to merge.

To learn more about rebasing in Git, there are a lot of
[good](http://git-scm.com/book/en/Git-Branching-Rebasing)
[resources](https://help.github.com/articles/interactive-rebase).

### 8. Merging a PR (maintainers only)

A PR can only be merged into master by a maintainer if:

* It is passing CI.
* It has been approved by at least one maintainer.
* It has no requested changes.
* It is up to date with current master.

Any maintainer is allowed to merge a PR if all of these conditions are
met.
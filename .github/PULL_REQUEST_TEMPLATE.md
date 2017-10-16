Type of change
==============

<!--- What types of changes does your code introduce? Put an `x` in all the boxes that apply: -->

- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)

Description
===========

<!--- What does this code solve? How does it solve it? -->

Review Checklist
================

<!--- Don't edit this, the reviewer will. Make sure you follow it though -->

Goals
-----

- [ ] Does it solve the problem?

- [ ] Is it the simplest implementation of that solution?
    Does it yak shave? Does it introduce new dependencies that aren't necessary?

- [ ] Does it decrease modularity?
    Does the user of a module need to import another module to use this one?
    If we want to delete these changes, how easy is that?

- [ ] Does it clarify our domain?
    What things does it refine? What things get added? How does this pave the way for new things?
    Are things named in such a way that a domain expert can find them?

- [ ] Does it introduce non-domain concepts?
    What does the user of this need to learn outside of our domain in order to use this?

Testing
-------

- [ ] Do we integration test changes to external services?

- [ ] Do we unit test code we can change?

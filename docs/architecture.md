# Architecture

## Overview

BitTorrent clients send Announce and Scrape requests to a _Frontend_.
Frontends parse requests and write responses for the particular protocol they implement.
The _TrackerLogic_ interface is used to generate responses for requests and optionally perform a task after responding to a client.
A configurable chain of _PreHook_ and _PostHook_ middleware is used to construct an instance of TrackerLogic.
PreHooks are middleware that are executed before the response has been written.
After all PreHooks have executed, any missing response fields that are required are filled by reading out of the configured implementation of the _Storage_ interface.
PostHooks are asynchronous tasks that occur after a response has been delivered to the client.
Because they are unnecessary to for generating a response, updates to the Storage for a particular request are done asynchronously in a PostHook.

## Diagram

![architecture diagram](https://user-images.githubusercontent.com/343539/52676700-05c45c80-2ef9-11e9-9887-8366008b4e7e.png)

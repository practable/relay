#!/bin/bash
docker run --env-file=book.env -p 4000:4000 relay_book:latest


# Heyluuk

Luuk at my [site](https://heylu.uk/)!

This project aims to:
* make fun of my first name
* publish my [othello bot](https://heylu.uk/at/dots) to the world
* _WIP_ create yet another link shortener
* _WIP_ publish predictions in a hashed way, to only be revealed in the future


## TODO

- [ ] Link Shortener
    - [x] create postgres
    - [x] create adminer
    - [x] create models
    - [x] try gorm migrations
    - [ ] create views
        - [x] redirecting
        - [ ] JSON API
            - [x] get: GET `/api/node/<id>`
            - [x] create: POST `/api/link` with JSON body
            - [ ] check: GET `/api/link/check
            - [ ] search: GET `/api/link/?q=query`
        - [ ] nice web UI
            - [ ] creating links
                - [x] basic form
                - [ ] check if link is available with ajax
                - [ ] feedback on submission
            - [ ] listing tree structure
            - [ ] searching
        - [ ] 404 page

- [ ] Predictions
    - [ ] create models
    - [ ] create views
        - [ ] listing
        - [ ] creating

- [ ] Good style
    - [ ] test coverage
    - [ ] CI
    - [ ] linting
    - [ ] log more in redirect handler
    - [x] use static jquery
    - [ ] human readable logs
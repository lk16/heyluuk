
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
            - [ ] get by parent id: GET `/api/node/?parent=<id>`
            - [x] root nodes: GET `/api/node/root`
            - [x] create: POST `/api/link` with JSON body
            - [ ] search: GET `/api/link/?q=query`
        - [ ] nice web UI
            - [ ] creating links
                - [x] basic form
                - [x] feedback on submission
                - [ ] fix bug: cannot post twice without refresh / get rid of recaptcha
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
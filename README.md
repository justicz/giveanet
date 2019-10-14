# giveanet.org

This is the code running [giveanet.org](https://giveanet.org/), a website I made to try and donate 10,000 mosquito nets to people in countries affected by malaria. Donations go directly to the [Against Malaria Foundation](https://againstmalaria.com), where 100% of the money is used to buy nets.

## Development Setup

### Build
```
docker-compose build
```
### Run
```
./util/start.sh
```

Integration tests will run automatically. Comment out the `testing` service in `docker-compose.yml` to prevent this.

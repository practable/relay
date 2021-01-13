# Relay

![alt text][logo]

![alt text][status]
![alt text][coverage]

Relay is a set of tools and services to let you to host remote lab experiments, without opening firewall ports.

 - Secure websocket relay and host adapter for sharing video and data, with read/write permissions
 - Secure login shell relay, host adapter and client  with end-to-end encrypted administration access without a jumpserver 
 - Booking server for connecting users to experiments
 - Works with experiments behind firewalls and NAT because all communications are relayed 
 - No need to open firewall ports, or get public IPv4 addresses.
 
## Background
 
Relay is the new core of the [practable.io](https://practable.io) remote laboratory ecosystem. Some of the educational thinking behind this ecosystem can be found [here](https://www.tandfonline.com/doi/full/10.1080/23752696.2020.1816845). If you cannot access the full-text of this paper (although please try, to show support for this idea), you can find an unformatted author final version in this repo at `docs/education`
  
To cite the paper:
>Timothy D. Drysdale, Simon Kelley, Anne-Marie Scott, Victoria Dishon, Andrew Weightman, Richard James Lewis & Stephen Watts (2020) Opinion piece: non-traditional practical work for traditional campuses, Higher Education Pedagogies, 5:1, 210-222, DOI: 10.1080/23752696.2020.1816845

## Status
 
 Relay v1.0 is being built to a February 2021 deadline and should be considered as having an unstable API until such time as this notice says otherwise.
  
  Having said that, the API is more or less in place now, and I expect to modify it only to suit immediate operational requirements that emerge in coming weeks.
  
  Once I've produced documentation to support my first batch of users, I will then improve this README to explain how it all hangs together from the point of view of developers - which let's face it, if you are here, you probably are. Meanwhile, feel free to browse in the repo to 'pkg/crossbar' and 'pkg/vw' where you can see (dated) introductions to key parts.
  
Just to match that theme of under-construction, here are some remote lab boxes part way through construction.
  
 ![alt text][boxes] 
  


[status]: https://img.shields.io/badge/status-development-yellow "status; development"
[coverage]: https://img.shields.io/badge/coverage-44%25-orange "Test coverage 44%"
[logo]: ./assets/images/logo.png "Relay ecosystem logo - hexagons connected in a network to a letter R"
[boxes]: ./assets/images/boxes-700x525.jpg "Boxes under construction"


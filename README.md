# SVOP
Simple VNF Orchestration Platform 

# Architecture

You can find the architecture in the link below:
![SVOP Architecture](./docs/svop-architecture.PNG)

## Problems Encountered

1. You should first ping the host on the overlay IP so that the rules are setup using the Ryu controller. Otherwise it won't work and SSH using overlay IP will fail.

#! /usr/bin/env sh

# Pass all the arguments of this script to the orb go executable
exec orb run -m ubuntu -u julien go "$@"

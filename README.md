# telenamer

Telenamer is a CLI tool that attempts to rename TV episode files to follow a consistent file name format. It does this by parsing the filename, pulling relevant information from it, and searching for the episode on TVDB to retrieve the full information about an episode.

## Installation

Either download a build from the "releases" section (only for Windows), or build it yourself with the following commands.

```bash
git clone https://github.com/arrivance/telenamer.git
go install github.com/arrivance/telenamer
```

## Usage

First of all you need to get your API key, User key and User name:

- Register an account on <http://thetvdb.com/?tab=register>
- When you are logged register an api key on <http://thetvdb.com/?tab=apiregister>
- View your api key, user key and user name on <http://thetvdb.com/?tab=userinfo>

To run, navigate to the directory with the episode, and type in:

```bash
telenamer --apikey "APIKEY" --userkey "USERKEY" --username "username"
```

(one can alternatively create a login.json file in the directory of the executable, in the format

```JSON
{
    "apikey": "APIKEY",
    "Userkey": "USERKEY",
    "Username": "USERNAME"
}
```

or anywhere else with the full path to the file passed with the -l parameter)

with the following optional parameters

- -f/--format: format the episode name
  - formatting syntax
    - {s}: series name
    - {n}: episode name
    - {z}/{0z}: season number ({0z} is 0-indexed for all season names less than 10)
    - {e}/{0e}: episode number ({0e} is 0-indexed for all episode names less than 10)
  - the default format is {s} - S{0z}E{0e} - {n}
- -u/--undo: performs an undo of the last operation.
- -s/--series: provide the series name if the filenames do not contain it.
- -c/--confirm: provide manual confirmation on every single file operation
- -z/--silent: provide no user output (does not work with -c)

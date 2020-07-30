# Request imports
import requests

# Pandas imports
import pandas as pd


# _COMMITS_URL is the GitHub API endpoint with recent Mutagen commit data.
_COMMITS_URL = 'https://api.github.com/repos/mutagen-io/mutagen/commits'


def load_commit_times():
    """Loads recent Mutagen commit times as a Pandas Series.
    """
    # Request commit data.
    commits = requests.get(_COMMITS_URL).json()

    # Extract commit times.
    times = [c['commit']['author']['date'] for c in commits]

    # Convert times to a Pandas series.
    return pd.Series(times).astype('datetime64')

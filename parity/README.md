# SAS parity comparison

Install Python dependencies with `python -m pip install -r parity/requirements.txt`.
From the repository root run:

```sh
python parity/compare.py /path/to/sas-files
```

The directory is searched recursively. The program builds the current checkout's
Go reader; use `--reader /path/to/dump` to compare a prebuilt alternative.
It compares raw metadata and values, plus converted dates for the named date
fixtures. Python conversion of `date_formats.sas7bdat` can overflow; the raw
comparison remains useful in that case. The Go API exposes format names, not
the format width or trailing dot. The harness recognizes matching invalid
Windows-1252 errors, but does not yet compare all error categories or every
pyreadstat date/time format.

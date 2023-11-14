Hi there! To use this, you will want something like this in your postgresql.conf:

    log_destionation = 'csvlog'
    logging_collector = on
    log_min_duration_statement = 0

This should put logs into .csv files in the log/ folder of your postgresql data folder.

Then, after compiling this and installing it into your $PATH, you can use it like this:

    tail -f postgresql-2023-11-14_122012.csv | pglogfmt

For much prettier logs!

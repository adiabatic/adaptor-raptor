# coding: UTF-8

# usage: sort.py [infile] [outfile]
# 
# infile and outfile can be the same file. Of course, there may be bugs.
# You’ve got infile in a version-control system, right? Then you’re good to go.

import yaml
import itertools
import codecs
import sys
import StringIO


def cmp_weird(lhs, rhs):
    return -cmp(lhs.lower(), rhs.lower())

def quote_maybe(s):
    if s in "yes no on off true false null .nan .inf".split():
        return "'{}'".format(s)
    elif '"' in s:
        return '{}'.format(s)
    elif "'" in s:
        return "{}".format(s)
    return s

if len(sys.argv) < 3:
    sys.stderr.write("sort.py infile outfile\n")
    sys.exit(1)

l = None
outs = StringIO.StringIO()
outsu = codecs.getwriter('UTF-8')(outs)
with open(sys.argv[1]) as f:
    l = list(yaml.load_all(f.read()))
    l.sort(cmp=cmp_weird, key=lambda e: e['f'])
    for lhs, rhs in itertools.combinations(l, 2):
        if 'f' in lhs and 'f' in rhs and lhs['f'] == rhs['f']:
            print "we got a duplicate:", lhs['f'], lhs

outsu.write("# This file is in the public domain.\n# http://creativecommons.org/publicdomain/zero/1.0/\n")
for e in l:
    outsu.write(u"---\n")
    if 'f' in e:
        outsu.write(u"f: {}\n".format(quote_maybe(e['f'])))
        del e['f']
    if 't' in e:
        outsu.write(u"t: {}\n".format(quote_maybe(e['t'])))
        del e['t']
    if len(e):
        outs.write(yaml.dump(e, encoding='UTF8', allow_unicode=True, default_flow_style=False))
    
with open(sys.argv[2], 'w') as f:
    x = outsu.getvalue()
    f.write(x)

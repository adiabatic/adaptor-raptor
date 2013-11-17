# coding: UTF-8

import yaml
import itertools
import codecs, locale, sys

#sys.stdout = codecs.getwriter('UTF-8')(sys.stdout)

def pr(s):
    sys.stdout.write(s.encode('u8')+'\n')

def cmp_weird(lhs, rhs):
    return -cmp(lhs.lower(), rhs.lower())

def by_to(lhs, rhs):
    return cmp

def quote_maybe(s):
    if s in "yes no on off true false null .nan .inf".split():
        return "'{}'".format(s)
    elif '"' in s:
        return '{}'.format(s)
    elif "'" in s:
        return "{}".format(s)
    return s

def chomp(s):
    if s[-1] == '\n': return s[:-1]
    return s
    

l = None
with open("50.yaml") as f:
    l = list(yaml.load_all(f.read()))
    l.sort(cmp=cmp_weird, key=lambda e: e['f'])
    for lhs, rhs in itertools.combinations(l, 2):
        if 'f' in lhs and 'f' in rhs and lhs['f'] == rhs['f']:
            print "we got a duplicate:", lhs['f'], lhs

for e in l:
    print "---"
    if 'f' in e:
        pr(u"f: {}".format(quote_maybe(e['f'])))
        del e['f']
    if 't' in e:
        pr(u"t: {}".format(quote_maybe(e['t'])))
        del e['t']
    if len(e):
        sys.stdout.write(chomp(yaml.dump(e, encoding='UTF8', allow_unicode=True, default_flow_style=False))+'\n')
    

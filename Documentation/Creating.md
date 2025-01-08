# Creating your own patterns

Try to find interesting patterns for yourself.

Select **File**->**New Tab** and it will create a blank slate.  You can
then enable **Edit Mode** and just add cells. You can click the **step
forward** and **step backward** buttons to explore how they evolve.

By default GooeyLife only saves 10 generations of history, but you can 
increase that in the **Settings** dialog.  Keep in mind, if you keep 
1000 generations of history on a large pattern like a _turing machine_
it will balloon the amount of memory that the program uses considerably.

Try this pattern:

![R Pentomino](images/R-Pentomino.png)

It's only got 5 cells, but will run for over 1000 generations (and 
spit out 5 _gliders_) before stagnating. This is sometimes called
a [Methuselah](https://conwaylife.com/wiki/Methuselah).

If you find an interesting pattern, you can save it using
**File**->**Save**, in a [RLE](https://conwaylife.com/wiki/Run_Length_Encoded)
file. (Be sure to end the filename with '.rle' or you'll
have trouble reloading it.)

## Other file formats

### Cells files

If you want to try creating patterns in a text editor, you can 
create `.cells` files which look like:

```
! Copperhead
! 'zdr'
! An c/10 orthogonal spaceship found on March 5, 2016.
! https://www.conwaylife.com/wiki/Copperhead
.OO..OO.
...OO...
...OO...
O.O..O.O
O......O
........
O......O
.OO..OO.
..OOOO..
........
...OO...
...OO...
```

Lines that begine with a `!` are considered comments, and the rest are a 
grid of `.` (dead cells) and `O` (live cells).

### Life files

I created my own file format early on in my exploration of the _Game of 
Life_ which I called `.life` files.  Unfortunatey, there are other standards
of `.life` files out there on the internet that are inconsistent with mine.

A life file looks like:

```
# Pulsar
# period 5
    **
   *  *
  *    *
 *      *
 *      *
  *    *
   *  *
    **
```

Anthing that begin with a `#` are comments, but othewise whitespace is
considered dead cells and anything else (other than whitespace and `#`)
is considered a live cell.  For instance, the _Gosper Glider Gun_ might 
look like:

```
# The Gosper glider gun
#   Discovered by Bill Gosper in November 1970
#   The first known infinitely growing pattern

                        B
                      B B
            AA      BB            **
           A   A    BB            **
**        A     A   BB
**        A   A AA    B B
          A     A       B
           A   A
            AA
```

I painstakingly typed this in my hand from a web illustration, and used `A`
and `B` to label what I thought were interesting structures.

This was all before I found that there are a [wealth
of interesting patterns on the web](Loading.md).

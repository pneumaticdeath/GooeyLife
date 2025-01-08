# Example Patterns

In the [Tour](Tour.md) we explored the simple examples of a _glider_ and
_blinker_, but now we'll explore some more interesting patterns.

The first is the the [Gosper Glider Gun](https://conwaylife.com/wiki/Gosper_glider_gun).
A question early on was whether all patterns eventually stagnated into simple oscilators,
static elements or would die out entirely.  That's what the inventor of the game, 
[John Conway](https://en.wikipedia.org/wiki/John_Horton_Conway) guessed, but offered a 
prize to anyone who could disprove his conjecture.  Soon (all the way back in 1970) 
a team at MIT lead by [Bill Gosper](https://en.wikipedia.org/wiki/Bill_Gosper) managed
to do that with the _glider gun_ pattern.  

Go to **Examples**->**Growing**->**Gosper Glider Gun**, and you'll load the pattern. It
doesn't look all that interesting, and only has 36 living cells initially, but if you click
**Run** you'll get a surprise.  At generation 15 (and every 30 thereafter), it spits out 
a _Glider_.  With nothing to stop them, they march off to infinity, and the pattern grows
indefinitely.

There are other patterns that grow indifintely. Like the **Block Laying Switch Engine** (also
under **Examples**->**Growing**).  For that one you may want to crank up the speed to about
60 generations/second (and make sure **Auto-Zoom** is enabled.)  This is strictly a type of 
Puffer, but we'll get to that in a moment.

There are other types too.  

## [Spaceships](https://conwaylife.com/wiki/Spaceship)

First there are the Spaceships.  Like Gliders they move across the grid.
Some (like the Lobster **Examples**->**Ships&Such**->**lobster.rle**) move
diagonally, while others like the **canadagrey.rle** ship move vertically
or horizontally. Relatively recently there has also been the discovery of
ships that move up 2 and over 1 (like the knight in chess), and so
they're called a [Knight Ship](https://conwaylife.com/wiki/Knightship).
The first was discovered in 2010 by Dave Greene and called **Sir Robin**.

## [Puffers](https://conwaylife.com/wiki/Puffer)

Puffers are spaceships that leave simple debris behind them. Take
for example the **Frothing Puffer**, which moves vertically, but leaves
a trail of blocks and blinkers in it's wake.

## [Rakes](https://conwaylife.com/wiki/Rake)

Rakes a puffers that spit out other spaceships, like the **Backrake 1**
which moves vertically, but spits out gliders.




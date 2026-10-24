# The Perceptron: The Simplest Neural Network

Neural networks can look mysterious from the outside, but they are built from one small part that is used many times.
The first version of that part is the perceptron. I programmed perceptrons in 2014, while still at university, and still
think it is the best place to start. To understand it, you need to add, multiply and compare numbers, nothing more.

This article explains the idea and walks through a short Go implementation:
<https://github.com/volodymyr-pavlov/go-perceptron>. The perceptron is the `neuron` package, and `make run` starts a
demo. Run it, change it, and try to break it.

## A Weighted Vote

Suppose you decide whether to go to a party. Three things matter: your friends will be there, it is far away, and you
have an exam tomorrow. They do not matter equally. Friends are a reason to go, the distance is a small reason not to go,
and the exam is a strong reason not to go.

A perceptron makes the same kind of decision with numbers. Every input $x_i$ gets a **weight** $w_i$. A big positive
weight means "this input matters, and it says yes". A negative weight means "this input says no". The perceptron adds up
the weighted inputs and adds a **bias** $b$, a fixed number that moves the total up or down. If the total is above zero,
the answer is 1. Otherwise, the answer is 0. With two inputs, the rule looks like this:

$$
y = 1 \text{ if } w_1 x_1 + w_2 x_2 + b > 0, \qquad y = 0 \text{ otherwise.}
$$

In Go, with any number of inputs, the whole decision is a loop and a comparison:

```go
func (p *Perceptron) Predict(x []float64) int {
    sum := p.bias
    for i, weight := range p.weights {
        sum += weight * x[i]
    }

    if sum > 0 {
        return 1
    }

    return 0
}
```

## A Straight Line

There is a picture behind the formula. Take two inputs and draw them as a point on a sheet of paper, with $x_1$ to the
right and $x_2$ up. The points where $w_1 x_1 + w_2 x_2 + b = 0$ form a straight line. On one side of the line, the sum
is positive and the perceptron says 1. On the other side, the sum is negative and it says 0. On the line itself, the sum
is zero, and the answer is 0.

So a perceptron splits a plane with one line. The weights change the angle of the line, and the bias moves it. Learning
means finding an angle and a position that put the points of the two answers on opposite sides.

## Learning From Mistakes

Nobody chooses the weights by hand. The perceptron starts with all the weights and the bias at zero, and it learns from
examples. For each example, it gives its answer $y$ and compares it with the right answer $t$. If they are the same,
nothing changes. If they are different, every weight moves a little towards the right answer:

$$
w_i \leftarrow w_i + \eta (t - y) x_i, \qquad b \leftarrow b + \eta (t - y).
$$

The number $\eta$ is the **learning rate**, the size of each step. The error $t - y$ is $+1$ when the answer should have
been 1, and $-1$ when it should have been 0. An input that is zero changes nothing, so only the weights of the inputs
that took part in the mistake change. One pass over all the examples is called an epoch. Here is the core of `Fit`,
which makes passes until one of them has no mistake:

```go
for epoch := 1; epoch <= maxEpochs; epoch++ {
    mistakes := 0

    for _, s := range samples {
        miss := s.Label - p.Predict(s.Inputs)
        if miss == 0 {
            continue
        }

        mistakes++

        step := p.rate * float64(miss)
        for i, value := range s.Inputs {
            p.weights[i] += step * value
        }

        p.bias += step
    }

    if mistakes == 0 {
        return epoch, nil
    }
}

return maxEpochs, ErrNotConverged
```

The version in the repository also checks the samples before it starts.

## Learning AND

Let us teach the perceptron AND: the answer is 1 only when both inputs are 1. The four examples are the rows of the
truth table.

| $x_1$ | $x_2$ | AND |
| --- | --- | --- |
| 0 | 0 | 0 |
| 0 | 1 | 0 |
| 1 | 0 | 0 |
| 1 | 1 | 1 |

With a learning rate of 0.1, the perceptron learns AND in six passes. Each row of the next table is one pass. It lists
the examples that got a wrong answer, in the order they came, with the error. It also shows the weights and the bias at
the end of the pass.

| Pass | Mistakes (inputs, error) | Weights after | Bias after |
| --- | --- | --- | --- |
| 1 | (1, 1) +1 | 0.1, 0.1 | 0.1 |
| 2 | (0, 0) −1, (0, 1) −1, (1, 1) +1 | 0.2, 0.1 | 0.0 |
| 3 | (0, 1) −1, (1, 0) −1, (1, 1) +1 | 0.2, 0.1 | −0.1 |
| 4 | (1, 0) −1, (1, 1) +1 | 0.2, 0.2 | −0.1 |
| 5 | (0, 1) −1 | 0.2, 0.1 | −0.2 |
| 6 | none | 0.2, 0.1 | −0.2 |

Pass 1 finds one mistake. For the input $(1, 1)$, the answer is 0, because all the weights and the bias are zero, but it
should be 1. So both weights and the bias grow by 0.1. In pass 2, the input $(0, 0)$ gives 1, because the bias alone is
now above zero, and the error $-1$ takes the bias back to 0. Pass 6 has no mistake, so the learning stops with weights
of 0.2 and 0.1 and a bias of $-0.2$. You can check the result. For $(1, 1)$, the sum is $0.2 + 0.1 - 0.2 = 0.1$, which
is above zero. For $(1, 0)$, the sum is exactly zero, which is not above zero, so the answer is 0, as it should be. For
the other two inputs, the sum is below zero.

Run `make run` in the repository to see this, and the same for OR. The output below is shortened to the summary lines:

```text
AND: learned in 6 passes
  weights [0.2 0.1], bias -0.2
OR: learned in 4 passes
  weights [0.1 0.1], bias 0.0
```

## Where It Fails: XOR

Now ask for XOR, "exclusive or": the answer is 1 when the two inputs are different.

```text
XOR: not learned in 100 passes, no straight line separates it
  weights [-0.1 0.0], bias 0.1
  [0 0] -> 1, should be 0: WRONG
  [0 1] -> 1, should be 1: right
  [1 0] -> 0, should be 1: WRONG
  [1 1] -> 0, should be 0: right
```

This is not a bug, and more passes do not help. Draw the four points on paper. The points $(0, 0)$ and $(1, 1)$ must get
0, and the points $(0, 1)$ and $(1, 0)$ must get 1. They are the corners of a square, and each pair is on one diagonal.
No single straight line has the first pair on one side and the second pair on the other side. A perceptron can only draw
a line, so it keeps fixing one mistake and creating another.

The opposite is also true. If a line that separates the examples exists, the learning rule always finds one, after a
limited number of steps. Albert Novikoff proved in 1962 that, starting from zero weights, the perceptron makes at most
$(R/\gamma)^2$ mistakes in total, at any learning rate. To state it exactly, add the bias to every input as one more
coordinate that is always 1. An input becomes $\mathbf{x} = (x_1, x_2, 1)$, and the weights with the bias become
$\mathbf{w} = (w_1, w_2, b)$. Let $s$ be $+1$ when the right answer is 1 and $-1$ when it is 0. Then

$$
R = \max \, |\mathbf{x}|, \qquad \gamma = \max_{|\mathbf{w}| = 1} \; \min \, s \, (\mathbf{w} \cdot \mathbf{x}),
$$

where the maximum and the minimum run over all the examples. So $R$ is the largest length of an input with the extra
coordinate, and $\gamma$ shows how well the best $\mathbf{w}$ of length 1 separates the examples. A line separates
them exactly when $\gamma > 0$. For AND, $R^2 = 3$ and $\gamma \approx 0.24$, so the limit is about 52 mistakes, and
the run above made 10. The test `TestSeparatesPointsOfALine` in the repository checks that the rule learns all 300
random points at three learning rates.

## What Came Next

Frank Rosenblatt introduced the perceptron in the late 1950s and built a machine that recognized simple shapes. In 1969,
Marvin Minsky and Seymour Papert described its limits, including XOR, in their book *Perceptrons*. After that, interest
in neural networks fell for many years.

The solution is simple: put perceptrons in layers. Several lines together can cut the plane into complicated shapes, and
XOR becomes easy. A layered network needs a way to teach all the layers at the same time. That method is called
backpropagation, and today's neural networks still use it. They are built from a similar part: a weighted sum followed
by a smooth function instead of the hard yes-or-no step, so that a small change of a weight changes the answer a little.
Millions or billions of these parts work together, with a more advanced learning rule.

## Try It Yourself

- Change the learning rate in `cmd/perceptron/main.go` to 1 and then to 0.001. Does AND still take six passes?
- Print the weights after each pass and watch the line move.
- Add a third input and teach the perceptron "at least two of the three inputs are 1". Change the number of inputs in
  `neuron.New` and the table of cases in `cmd/perceptron/main.go`.
- Look for a set of four points, two of each answer, that a line cannot split. Is XOR the only one?

The answer to the first question is yes: it takes six passes at every rate, and the weights only become larger or
smaller with the rate. The perceptron starts at zero, and every step is a multiple of the rate, so the rate changes the
size of the numbers but never the sign of a sum. The test `TestRateOnlyScalesTheWeights` in the repository checks this
for 1, 0.1 and 0.001. If the starting weight is not zero, the rate would matter.

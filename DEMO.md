# xyztree demo

A demo involving all the package's functions where we
* pick 100,000 random points from the unit cube,
* create the corresponding 3-d tree,
* save the result to the JSON file "demo.json",
* read the 3-d tree back from the above file and
* test nearest-neighbord searching against brute-force searching using the Euclidean metric for 1,000 random points.

```go
package main

import(
    "fmt"
    "github.com/ybeaudoin/go-xyztree"
    "math"
    "math/rand"
    "os"
    "strconv"
    "time"
)

func main() {
    const(
          numPts   = 100000
          numTests = 1000
    )
    //Pick random points from the unit cube
    rand.Seed(time.Now().UnixNano())
    points := make(xyztree.DataSet, numPts)
    for ptNo := 1; ptNo <= numPts; ptNo++ {
        points["#" + strconv.Itoa(ptNo)] = xyztree.DataCoords{rand.Float64(), rand.Float64(), rand.Float64()}
    }
    fmt.Println("- data created")

    //Create the corresponding 3-d tree
    xyztree.Make(&points, false)
    fmt.Println("- 3-d tree created")

    //Save the result to a JSON file (indented mode)
    xyztree.Export("demo.json", false)
    fmt.Println("- 3-d tree exported")

    //Read the 3-d tree back
    xyztree.Import("demo.json")
    fmt.Println("- 3-d tree imported")

    //Test nearest-neighbord search against brute-force search using the Euclidean metric
    for testNo := 1; testNo <= numTests; testNo++ {
        vTest := xyztree.DataCoords{rand.Float64(), rand.Float64(), rand.Float64()}
        fmt.Println("\ntest   : pt =", vTest)

        //using 3-d tree
        nnNode  := xyztree.NN(&vTest, "Euclidean", false)
        v1      := nnNode.COORDS
        fmt.Printf("xyztree: pt = %v, key = %s\n", v1, nnNode.KEY)

        //using brute force
        bestKey    := ""
        bestMetric := math.MaxFloat64
        for k, v := range points {
            metric := calcEuclidean(&vTest, &v)
            if metric < bestMetric { bestKey, bestMetric = k, metric }
        }
        v2 := points[bestKey]
        fmt.Printf("brute  : pt = %v, key = %s\n", v2, bestKey)

        //check for failure: if keys differ, check for unequal distances to rule out more than one best match
        if (nnNode.KEY != bestKey) && (calcEuclidean(&vTest, &v1) != calcEuclidean(&vTest, &v2)) {
            fmt.Println("\aFAIL")
            os.Exit(0)
        }
        fmt.Println("PASS")
    }
    fmt.Println("\nSUCCESS!")
}
func calcEuclidean(pt1, pt2 *xyztree.DataCoords) float64 {
    diff := (*pt1)[0] - (*pt2)[0]; metric := diff * diff
    diff  = (*pt1)[1] - (*pt2)[1]; metric += diff * diff
    diff  = (*pt1)[2] - (*pt2)[2]; metric += diff * diff
    return math.Sqrt(metric)
}
```
















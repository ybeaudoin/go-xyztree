/*===== Copyright (c) 2016 Yves Beaudoin - All rights reserved - MIT LICENSE (MIT) - Email: webpraxis@gmail.com ================
 * Package:
 *     xyztree
 * Overview:
 *     package for creating, exporting, importing and nearest-neighbor searching 3-d trees.
 * Types:
 *     DataCoords
 *         array for a data point's float64 R^3 coordinates, i.e., [1.,2.,3.]
 *     DataSet
 *         map for the R^3 data points keyed on string identifiers, i.e., "Pt1":[1.,2.,3.], "Pt2":[4.,5.,6.], etc.
 *     Node
 *         structure for a 3-d tree node
 * Functions:
 *     Export(file string, compact bool)
 *         Exports the 3-d tree to a specified file using the JSON format with or without newlines and identations.
 *     Import(file string)
 *         Imports a 3-d tree from a specified JSON file.
 *     Make(refPoints *DataSet, verbose bool)
 *         Creates a 3-d tree recursively.
 *     NN(refTestPt *DataCoords, metric string, verbose bool) (nnNode Node)
 *         Finds the nearest neighbor of a test point using a specified metric. Returns a best-matching 3-d tree node.
 * Remarks:
 *     The 3-d tree is a slice of structures constituting a top-down binary node list. Each slice element stores a node's
 *     particulars as follows:
 *      HYPERPLANE => integer axis index in the range [0,2] for the hyperplane: a value of -1 indicates a leaf node.
 *      KEY        => string for the node point identifier.
 *      COORDS     => array for the node point coordinates.
 *      LEFTCHILD  => integer link to the left child node: a value of -1 indicates no child node.
 *      RIGHTCHILD => integer link to the right child node: a value of -1 indicates no child node.
 * History:
 *     v1.1.0 - October 31, 2016 - Moded JSON import & export for empty LEFTCHILD and RIGHTCHILD fields
 *     v1.0.0 - October 19, 2016 - Original release
 *============================================================================================================================*/
package xyztree

import(
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "math"
    "os"
    "runtime"
    "sort"
    "strings"
)
/*Exported -------------------------------------------------------------------------------------------------------------------*/
type(
    DataCoords [3]float64            //array for a data point's R^3 coordinates
    DataSet    map[string]DataCoords //map for the R^3 data points keyed on identifiers
    Node struct {                    //3-d tree node structure:
        HYPERPLANE int               // axis index for the hyperplane or -1 if leaf node
        KEY        string            // point identifier
        COORDS     DataCoords        // array for the point coordinates
        LEFTCHILD  int               // link to the corresponding left child node or -1 if no child
        RIGHTCHILD int               // link to the corresponding right child node or -1 if no child
    }
)

func Export(file string, compact bool) {
/*         Purpose : Exports the 3-d tree to a specified file using the JSON format with or without newlines and identations.
 *       Arguments : file    = filename for the JSON output,
 *                   compact = boolean flag for compact mode.
 *         Returns : None.
 * Externals -  In : _3dtree, jsonNode, jsonTree
 * Externals - Out : None.
 *       Functions : halt
 *         Remarks : None.
 *         History : v1.1.0 - October 31, 2016 - Moded to omit empty LEFTCHILD and RIGHTCHILD fields
 *                   v1.0.0 - October 13, 2016 - Original release.
 */
    if len(_3dtree) == 0 { halt("there's no 3-d tree to export") }

    var(
        numNodes = len(_3dtree)
        output   []byte
    )

    writer, err := os.Create(file) //open file for write
    if err != nil { halt("os.Create - " + err.Error()) }
    defer writer.Close()

    nodeData := make([]jsonNode, numNodes)
    for k, v := range _3dtree {
        if v.HYPERPLANE == -1 {
            v.LEFTCHILD, v.RIGHTCHILD = 0, 0
        }
        nodeData[k] = jsonNode{ ID:         k,
                                HYPERPLANE: v.HYPERPLANE,
                                KEY:        v.KEY,
                                COORDS:     v.COORDS,
                                LEFTCHILD:  v.LEFTCHILD,
                                RIGHTCHILD: v.RIGHTCHILD }
    }
    jsonData := jsonTree{ SIZE:    numNodes,
                          XYZTREE: nodeData }
    if compact { output, err = json.Marshal(jsonData) // create the JSON output
    } else     { output, err = json.MarshalIndent(jsonData, "", " ") }
    if err != nil { halt("Marshal/MarshalIndent - " + err.Error()) }

    _, err = writer.Write(output) //save the tree
    if err != nil { halt("writer.Write - " + err.Error()) }
    if err = writer.Sync(); err != nil { halt("writer.Sync - " + err.Error()) }
    if err = writer.Close(); err != nil { halt("writer.Close - " + err.Error()) }
    return
} //end func Export
func Import(file string) {
/*         Purpose : Imports a 3-d tree from a specified JSON file.
 *       Arguments : file = data filename.
 *         Returns : None.
 * Externals -  In : node
 * Externals - Out : _3dtree, jsonTree
 *       Functions : halt
 *         Remarks : None.
 *         History : v1.1.0 - October 31, 2016 - Moded for empty LEFTCHILD and RIGHTCHILD fields
 *                   v1.0.0 - October 13, 2016 - Original release.
 */
    if fi, err := os.Stat(file); (err != nil) || (fi.Size() == 0) {
        halt("the input file cannot be located or is empty")
    }

    var jsonIn jsonTree

    input, err := ioutil.ReadFile(file) //read the whole file
    if err != nil { halt("ioutil.ReadFile - " + err.Error()) }

    err = json.Unmarshal(input, &jsonIn) // decode the JSON data
    if err != nil { halt("json.Unmarshal - " + err.Error()) }

    _3dtree = make([]Node, jsonIn.SIZE, jsonIn.SIZE)
    for _, v := range jsonIn.XYZTREE {
        if v.HYPERPLANE == -1 {
            _3dtree[v.ID] = Node{v.HYPERPLANE, v.KEY, v.COORDS, -1, -1}
        } else {
            _3dtree[v.ID] = Node{v.HYPERPLANE, v.KEY, v.COORDS, v.LEFTCHILD, v.RIGHTCHILD}
        }
    }
    return
} //end func Import
func Make(refPoints *DataSet, verbose bool) {
/*         Purpose : Creates a 3-d tree recursively.
 *       Arguments : refPoints = reference to the map of float64 data points in R^3, keyed on string identifiers.
 *                   verbose   = boolean flag for verbose mode. If true, the main execution stages will be echoed to Stdout.
 *         Returns : None.
 * Externals -  In : DataSet
 * Externals - Out : _3dtree, _builder
 *       Functions : halt, makeBuilder
 *         Remarks : None.
 *         History : v1.0.0 - October 13, 2016 - Original release.
 */
    if len(*refPoints) == 0 { halt("there are no points to process") }

    _3dtree  = nil                  //clear the 3-d tree
    _builder = makeBuilder(verbose) //make recursive tree builder

    _builder(refPoints)
    return
} //end func Make
func NN(refTestPt *DataCoords, metric string, verbose bool) (nnNode Node) {
/*         Purpose : Finds the nearest neighbor of a test point using a specified metric.
 *       Arguments : refTestPt = reference to the R^3 coordinates of the test point.
 *                   metric    = distance metric: 'Euclidean', 'Manhattan' or 'Max'.
 *                   verbose   = boolean flag for verbose mode. If true, the main execution stages will be echoed to Stdout.
 *         Returns : nnNode    = a 3-d tree node with closest data point.
 * Externals -  In : DataCoords, _3dtree
 * Externals - Out : _nn
 *       Functions : halt, makeFinder
 *         Remarks : None.
 *         History : v1.0.0 - October 19, 2016 - Original release.
 */
    if len(_3dtree) == 0 { halt("there's no 3-d tree to search") }

    if verbose { fmt.Println("test pt: ", *refTestPt) }
    _nn    = makeFinder(metric, verbose) //make recursive nn finder
    nnNode = _nn(0, refTestPt)           //start search at the root node
    if verbose { fmt.Printf("Nearest Neighbor key: %s\n\n", nnNode.KEY) }
    return
} //end func NN
/*Private --------------------------------------------------------------------------------------------------------------------*/
type(
    builderFn      func(refPoints *DataSet)                      //3-d tree builder
    finderFn       func(nodeIdx int, refTestPt *DataCoords) Node //nearest neighbor finder
    metricFn       func(pt1, pt2 *DataCoords) float64            //distance metric

    jsonNode struct {                             //JSON structure for a 3-d tree node:
        ID         int        `json:"id"`         // node meta data
        HYPERPLANE int        `json:"hyperplane"` // node data
        KEY        string     `json:"key"`
        COORDS     DataCoords `json:"coords"`
        LEFTCHILD  int        `json:"leftchild,omitempty"`
        RIGHTCHILD int        `json:"rightchild,omitempty"`
    }
    jsonTree struct {                             //JSON structure for the 3-d tree:
        SIZE       int        `json:"size"`
        XYZTREE    []jsonNode `json:"xyztree"`
    }
    ptPair struct {                               //structure for sorting the points:
        KEY        string                         // point identifier
        COORDS     DataCoords                     // array for the point coordinates
    }
    ptList         []ptPair                       //array for point identifiers & coordinates
    spanPair struct {                             //structure for sorting the axis lengths:
        AXIS       int                            // axis index in the range [0,2]
        LENGTH     float64                        // axis span
    }
    spanList       []spanPair                     //array for the axis indexes & lengths
)
var(
    _3dtree        []Node                         //3-d tree as a slice of nodes
    _builder       builderFn                      //3-d tree builder
    _nn            finderFn                       //nearest-neighbor finder

    _hyperplane    int                            //axis index in the range [0,2] used to partition the points using medians
    _eor           = -1                           //end-of-recursion marker indicating a leaf or absent branch node
)
////3-d tree build
func makeBuilder(argVerbose bool) builderFn {
    var(
        masterNodeIdx int          //master node index
        verbose       = argVerbose //verbose mode
        verboseIndent int          //number of spaces to indent when verbose
    )
    return func(refPoints *DataSet) {
            var(
                medianKey string            //identifier for the median point
                medianPt  DataCoords        //coordinates of the median point
                leftPts   = make(DataSet)   //points to the "left"
                rightPts  = make(DataSet)   //points to the "right"
                numPts    = len(*refPoints) //number of data points
            )
            //Initialize
            _3dtree      = append(_3dtree, Node{}) //add blank node to 3-d tree
            thisNodeIdx := masterNodeIdx           //set this node's index value
            //Check for a leaf node
            if numPts == 1 {
                for k, v := range *refPoints {
                   _3dtree[thisNodeIdx] = Node{_eor, k, v, _eor, _eor}
                   if verbose { fmt.Printf("%sNODE #%d: leaf, key = %s\n", strings.Repeat(" ",verboseIndent), thisNodeIdx, k) }
                }
                return
            }
            //Do a binary space partition
            medianKey, medianPt, leftPts, rightPts = bsp(refPoints)
            _3dtree[thisNodeIdx].HYPERPLANE        = _hyperplane
            _3dtree[thisNodeIdx].KEY               = medianKey
            _3dtree[thisNodeIdx].COORDS            = medianPt
            if verbose {fmt.Printf("%sNODE #%d: numPts = %v, median key = %s, median pt = %v, hyperplane = %v\n",
                                   strings.Repeat(" ",verboseIndent), thisNodeIdx, numPts, medianKey, medianPt, _hyperplane)}
            //Create the child nodes
            verboseIndent++                                     //update verbose indent
            if len(leftPts) > 0 {                               //create left child:
                masterNodeIdx++                                 // update master node index
                _3dtree[thisNodeIdx].LEFTCHILD = masterNodeIdx  // link parent to child node
                _builder(&leftPts)                              // add child node to 3-d tree
            } else {
                _3dtree[thisNodeIdx].LEFTCHILD = _eor           // indicate no left child
            }
            if len(rightPts) > 0 {                              //create right child:
                masterNodeIdx++                                 // update master node index
                _3dtree[thisNodeIdx].RIGHTCHILD = masterNodeIdx // link parent to child node
                _builder(&rightPts)                             // add child node to 3-d tree
            } else {
                _3dtree[thisNodeIdx].RIGHTCHILD = _eor          // indicate no right child
            }
            verboseIndent--                                     //update verbose indent
            return
           }
} //end func makeBuilder
func (p ptList)   Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p ptList)   Len() int { return len(p) }
func (p ptList)   Less(i, j int) bool { return p[i].COORDS[_hyperplane] < p[j].COORDS[_hyperplane] }
func (p spanList) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p spanList) Len() int { return len(p) }
func (p spanList) Less(i, j int) bool { return p[i].LENGTH < p[j].LENGTH }

func bsp(refPoints *DataSet) (medianKey string, medianPt DataCoords, leftPts, rightPts DataSet) {
    type(
        minMax struct {
            MIN, MAX float64
        }
    )
    var(
        dataBounds [3]minMax
        numPts     = len(*refPoints)
        medianIdx  = int(math.Trunc(float64(numPts)/2.))
        points     = make(ptList, numPts)
        spans      = make(spanList, 3)
    )
    //Split along the longest axis of the data set in order to spread the points out more evenly
    for k := range dataBounds { //initialize
        dataBounds[k] = minMax{ math.Inf(1), math.Inf(-1) }
    }
    for _, v := range *refPoints { //compute min & max for each axis
        for k := range dataBounds {
            dataBounds[k].MIN = math.Min(v[k], dataBounds[k].MIN)
            dataBounds[k].MAX = math.Max(v[k], dataBounds[k].MAX)
        }
    }
    for k := range dataBounds { //find longest axis length
        spans[k] = spanPair{k, dataBounds[k].MAX - dataBounds[k].MIN }
    }
    sort.Sort(sort.Reverse(spans))
    _hyperplane = spans[0].AXIS //set hyperplane index
    //Sort the points by their "hyperplane" component
    i := 0
    for k, v := range *refPoints {
        points[i] = ptPair{k, v}
        i++
    }
    sort.Sort(points)
    //Use the median as the partition point
    medianKey = points[medianIdx].KEY
    medianPt  = points[medianIdx].COORDS
    leftPts   = make(DataSet)
    rightPts  = make(DataSet)
    for k := 0; k < medianIdx; k++ {
        leftPts[points[k].KEY] = points[k].COORDS
    }
    for k := medianIdx + 1; k < numPts; k++ {
        rightPts[points[k].KEY] = points[k].COORDS
    }
    return
} //end func bsp
////Nearest Neighbor
func makeFinder(argMetric string, argVerbose bool) finderFn {
    var(
        metric        = makeMetric(argMetric) //distance metric
        verbose       = argVerbose            //verbose mode
        nnDist        = math.Inf(1)           //distance between the current nearest neighbor and the test point
        nnIdx         int                     //3-d tree node index with the current nearest-neighbor point
        verboseIndent int                     //number of spaces to indent when verbose
    )
    return func(nodeIdx int, refTestPt *DataCoords) Node {
            //Parameterize the node data
            var(
                hyperplane = _3dtree[nodeIdx].HYPERPLANE
                nodeKey    = _3dtree[nodeIdx].KEY
                nodeCoords = _3dtree[nodeIdx].COORDS
                leftChild  = _3dtree[nodeIdx].LEFTCHILD
                rightChild = _3dtree[nodeIdx].RIGHTCHILD
            )
            //Update the best match
            nodeDist := metric(refTestPt, &nodeCoords)
            if nodeDist < nnDist {
                nnIdx, nnDist = nodeIdx, nodeDist
            }
            if verbose { fmt.Printf("%sNODE #%d: key = %s, distance = %v\n",
                                    strings.Repeat(" ",verboseIndent), nodeIdx, nodeKey, nodeDist) }
            //Recurse thru child nodes if need be
            verboseIndent++                                                                   //update verbose indent
            switch {                                                                          //what next?
                case (nnDist == 0.) || (hyperplane == _eor):                                  // exact match or leaf node
                case leftChild == _eor:                                                       // just a right child
                    _nn(rightChild, refTestPt)
                case rightChild == _eor:                                                      // just a left child
                    _nn(leftChild, refTestPt)
                case (*refTestPt)[hyperplane] < nodeCoords[hyperplane]:                       // go to left child
                    _nn(leftChild, refTestPt)
                    if math.Abs((*refTestPt)[hyperplane] - nodeCoords[hyperplane]) < nnDist { //  need to go to right child?
                        _nn(rightChild, refTestPt)
                    }
                default:                                                                      // go to right child
                    _nn(rightChild, refTestPt)
                    if math.Abs((*refTestPt)[hyperplane] - nodeCoords[hyperplane]) < nnDist { //  need to go to left child?
                        _nn(leftChild, refTestPt)
                    }
            }
            verboseIndent--                                                                   //update verbose indent
            return _3dtree[nnIdx]
           }
} //end func makeFinder
func makeMetric(metricName string) (metric metricFn) {
    switch metricName {
        case "Euclidean":
            metric = func(pt1, pt2 *DataCoords) float64 {
                      diff := (*pt1)[0] - (*pt2)[0]; metric := diff * diff
                      diff  = (*pt1)[1] - (*pt2)[1]; metric += diff * diff
                      diff  = (*pt1)[2] - (*pt2)[2]; metric += diff * diff
                      return math.Sqrt(metric)
                     }
        case "Manhattan":
            metric = func(pt1, pt2 *DataCoords) float64 {
                      return math.Abs((*pt1)[0] - (*pt2)[0]) +
                             math.Abs((*pt1)[1] - (*pt2)[1]) +
                             math.Abs((*pt1)[2] - (*pt2)[2])
                     }
        case "Max":
            metric = func(pt1, pt2 *DataCoords) float64 {
                      return math.Max(math.Abs((*pt1)[0] - (*pt2)[0]),
                                      math.Max(math.Abs((*pt1)[1] - (*pt2)[1]),
                                               math.Abs((*pt1)[2] - (*pt2)[2])))
                     }
        default:
            halt("unrecognized metric name '" + metricName + "'")
    }
    return
} //end fun makeMetric
////Reporting
func halt(msg string) {
    pc, _, _, ok := runtime.Caller(1)
    details      := runtime.FuncForPC(pc)
    if ok && details != nil {
        log.Fatalln(fmt.Sprintf("\a%s: %s", details.Name(), msg))
    }
    log.Fatalln("\axyztree: FATAL ERROR!")
} //end func halt
//===== Copyright (c) 2016 Yves Beaudoin - All rights reserved - MIT LICENSE (MIT) - Email: webpraxis@gmail.com ================
//end of Package xyztree

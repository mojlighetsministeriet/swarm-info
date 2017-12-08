const links = []
const nodes = [{ id: "swarm", group: 0, label: "Swarm" }]

const svg = d3.select("svg"),
    width = +svg.attr("width"),
    height = +svg.attr("height"),
    color = d3.scaleOrdinal(d3.schemeCategory10);

const simulation = d3.forceSimulation(nodes)
    .force("charge", d3.forceManyBody().strength(-1000))
    .force("link", d3.forceLink(links).distance(200))
    .force("x", d3.forceX())
    .force("y", d3.forceY())
    .alphaTarget(1)
    .on("tick", ticked);

const rootGroup = svg.append("g").attr("transform", "translate(" + width / 2 + "," + height / 2 + ")");
let linkGroup = rootGroup.append("g").attr("stroke", "#fff").attr("stroke-width", 1.5).selectAll(".link");
let nodeGroup = rootGroup.append("g").attr("stroke", "#fff").attr("stroke-width", 1.5).selectAll(".node");
let labelGroup = rootGroup.append("g").attr("fill", "#fff").selectAll(".label");

restart();

function restart() {

  // Apply the general update pattern to the nodes.
  nodeGroup = nodeGroup.data(nodes, (node) => { return node.id;});
  nodeGroup.exit().remove();
  nodeGroup = nodeGroup.enter().append("circle")
    .attr("fill", (node) => {
      if (node.container) {
        return 'yellow';
      } else if (node.node) {
        return 'orange';
      } else {
        return 'brown';
      }
    })
    .attr("r", 15)
    .merge(nodeGroup);

  // Apply the general update pattern to the labels.
  labelGroup = labelGroup.data(nodes, (node) => { return node.id; });
  labelGroup.exit().remove();
  labelGroup = labelGroup.enter().append('text')
    .text((node) => { return node.label; })
    .attr('dx', 15)
    .attr('dy', 15)

  // Apply the general update pattern to the links.
  linkGroup = linkGroup.data(links, (link) => { return link.source.id + "-" + link.target.id; });
  linkGroup.exit().remove();
  linkGroup = linkGroup.enter().append("line").merge(linkGroup);

  // Update and restart the simulation.
  simulation.nodes(nodes);
  simulation.force("link").links(links);
  simulation.alpha(1).restart();
}

function ticked() {
  nodeGroup
    .attr("cx", (node) => { return node.x; })
    .attr("cy", (node) => { return node.y; })

  labelGroup
    .attr('x', function (node) { return node.x })
    .attr('y', function (node) { return node.y });

  linkGroup
    .attr("x1", (link) => { return link.source.x; })
    .attr("y1", (link) => { return link.source.y; })
    .attr("x2", (link) => { return link.target.x; })
    .attr("y2", (link) => { return link.target.y; });
}

// diffing and mutating the data
function updateData() {
  d3.json('/api/aggregate/', (error, data) => {
    if (error) {
      console.error(error);
      setTimeout(updateData, 3000);
      return;
    }

    const newNodes = [];

    data.nodes.forEach((node) => {
      newNodes.push({label: node.hostname, id: node.id, group: 'nodes', node, radius: 15});
      node.containers.forEach((container) => {
        newNodes.push({label: container.name, id: container.id, group: 'containers', container, radius: 10});
      });
    });

    const diff = {
      removed: nodes.filter((node) => {
        let inNewNodes = false
        newNodes.forEach((newNode) => {
          if (node.id === newNode.id) {
            inNewNodes = true
          }
        })

        return !inNewNodes && node.id !== 'swarm';
      }),
      added: newNodes.filter((newNode) => {
        let inNodes = false
        nodes.forEach((node) => {
          if (newNode.id === node.id) {
            inNodes = true
          }
        })

        return !inNodes;
      })
    }

    if (diff.removed.length || diff.added.length) {
      diff.removed.forEach((node) => {
        nodes.splice(nodes.indexOf(node), 1)
        for (const i = links.length; i >= 0; i--) {
          const link = links[i]
          if (link && (node.id === link.source.id || node.id === link.target.id)) {
            links.splice(i, 1)
          }
        }
      })

      diff.added.forEach((node) => {nodes.push(node)})
      diff.added.forEach((node) => {
        if (node.container) {
          let source;
          nodes.forEach((sourceCandidate) => {
            if (sourceCandidate.id === node.container.nodeId) {
              source = sourceCandidate
            }
          })
          if (source) {
            links.push({source, target: node, strength: 25})
          }
        } else if (node.node) {
          links.push({source: nodes[0], target: node, strength: 50})
        }
      })

      restart();
    }

    setTimeout(updateData, 3000);
  })
}
updateData();

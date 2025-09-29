import networkx as nx
import matplotlib.pyplot as plt
import matplotlib.animation as animation
import random

# Set seed for reproducible results
random.seed(42)

# 1. SETUP THE NETWORK
# Create a graph with 5 nodes
G = nx.Graph()
nodes = range(5)
G.add_nodes_from(nodes)

# Add edges to connect the nodes
# You can change these connections to create a different network structure
edges = [(0, 1), (0, 2), (1, 3), (2, 4), (3, 4)]
G.add_edges_from(edges)

# Define the starting node for the gossip
start_node = 0
# This set will keep track of nodes that have heard the gossip
informed_nodes = {start_node}
# Keep track of when each node was informed
inform_step = {start_node: 0}

# Use a fixed layout for nodes so they don't move around
pos = nx.spring_layout(G, seed=42)

# 2. SETUP THE PLOT FOR ANIMATION
fig, ax = plt.subplots(figsize=(7, 7))

def update(frame):
    """
    This function is called for each frame of the animation.
    It updates which nodes are 'informed' and redraws the network.
    """
    ax.clear() # Clear the previous frame to prepare for the new one

    # --- Gossip Spreading Logic ---
    # Only spread gossip after frame 0 (initial state)
    if frame > 0:
        # Find nodes that were informed in the previous step
        nodes_to_spread_from = [node for node, step in inform_step.items() if step == frame - 1]

        # Each informed node randomly selects one neighbor to share gossip with
        for node in nodes_to_spread_from:
            uninformed_neighbors = [n for n in G.neighbors(node) if n not in informed_nodes]
            if uninformed_neighbors:
                # Randomly select one neighbor to inform
                chosen_neighbor = random.choice(uninformed_neighbors)
                informed_nodes.add(chosen_neighbor)
                inform_step[chosen_neighbor] = frame

    # 3. DRAW THE NETWORK FOR THE CURRENT FRAME
    # Assign colors: 'red' if informed, a shade of blue if not
    node_colors = ['red' if node in informed_nodes else '#6495ED' for node in G.nodes()]

    # Draw the nodes, edges, and labels
    nx.draw(
        G,
        pos,
        ax=ax,
        with_labels=True,
        node_color=node_colors,
        node_size=1000,
        font_color='white',
        font_size=16,
        edge_color='gray'
    )

    # Set a title for the plot showing the current animation step
    ax.set_title(f"Gossip Spread: Step {frame}", fontweight="bold")
    ax.set_axis_off() # Turn off the axis for a cleaner look

# 4. CREATE AND SAVE THE ANIMATION
# This object drives the animation by calling the 'update' function for each frame
ani = animation.FuncAnimation(fig, update, frames=6, interval=1000, repeat=False)

# --- Save the animation to a file ---
# The script will print messages to the console to show its progress
print("Saving animation as gossip_spread.gif... This might take a moment.")
ani.save("gossip_spread.gif", writer="pillow", fps=1)
print("Animation saved successfully!")

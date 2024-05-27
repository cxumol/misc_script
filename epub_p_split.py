#pip install EbookLib beautifulsoup4
#cut-epub.ipynb https://colab.research.google.com/drive/17fWNb_KLh1X1gawcO1EVqY4H5t_r-MPT
path = 'myfile.epub'
import ebooklib
from ebooklib import epub
from bs4 import BeautifulSoup

path = 'hazelyoung-Katalepsis.epub'
num_parts = 10  # Change this variable to divide into X parts

# Load the EPUB file
book = epub.read_epub(path)

# Iterate over each item in the EPUB
for item in book.get_items():
    # Check if the item is an HTML file
    if item.get_type() == ebooklib.ITEM_DOCUMENT:
        # Parse the HTML content with BeautifulSoup
        soup = BeautifulSoup(item.get_content(), 'html.parser')

        # Find all <p> tags
        p_tags = soup.find_all('p')

        # Iterate over each <p> tag
        for p_tag in p_tags:
            # Get the text content of the <p> tag
            p_text = p_tag.get_text(separator='\n')

            # Check if the text length exceeds 3000 characters
            if len(p_text) > 3000:
                # Split the text into X approximately equal parts
                print("split before", len(p_text))
                split_points = []
                for i in range(1, num_parts):
                    split_points.append(p_text.rfind('.', 0, len(p_text) * i // num_parts))

                # Replace the original <p> tag with X new <p> tags
                new_p_tags = [soup.new_tag('p', attrs={'class': p_tag.get('class', [])}) for _ in range(num_parts)]

                # Append the split text to the new <p> tags
                for i in range(num_parts):
                    if i == 0:
                        new_p_tags[i].append(p_text[:split_points[0]+1])
                    elif i == num_parts - 1:
                        new_p_tags[i].append(p_text[split_points[i-1]+1:])
                    else:
                        new_p_tags[i].append(p_text[split_points[i-1]+1:split_points[i]+1])
                print("split after", [len(x.get_text()) for x in new_p_tags])

                # Replace the original <p> tag with the new <p> tags
                p_tag.replace_with(*new_p_tags)
            else:
                # If the paragraph is not too long, leave it as is
                pass

        # Update the content of the item with the modified HTML
        item.set_content(str(soup))

# Save the modified EPUB file
epub.write_epub(f'modified_{path}', book)